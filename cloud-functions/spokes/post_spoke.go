package spokes

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/spokes/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/dbutil"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	commonModels "github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"net/http"
	"os"
	"time"
)

var postSpokePath = urit.MustCreateTemplate(fmt.Sprintf("/sites/{%s}/spokes", common.PathParamSiteID))

func init() {
	functions.HTTP("PostSpoke", postSpoke)
}

func postSpoke(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("post_spoke.postSpoke"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	postSpokeHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()),
		cloud.NewPubSubRepository(requestWithContext.Context()))
}

func postSpokeHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("post_spoke.postSpokeHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	var spoke models.Spoke
	var siteSpoke models.SiteSpoke
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: models.GetRequiredHeaders(),
		RequiredPath:    postSpokePath,
		RequestMethod:   http.MethodPost,
		RequestBodyValidation: &utils.RequestBodyValidation{
			Entity:             &spoke,
			CompleteValidation: true,
		},
	})
	if validationResponse != nil {
		logger.Errorf("Request body validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	siteID := pathParams[common.PathParamSiteID]
	retailerID := request.Header.Get(common.HeaderRetailerID)
	var err error

	if !dbutil.IsRetailerIDPresentInDB(responseWriter, request, dbClient, retailerID, logger, true) {
		return
	}

	if !dbutil.IsSiteIDPresentInDB(responseWriter, request, dbClient, retailerID, siteID, logger, true) {
		return
	}

	existsSpokeName, err := dbClient.Exists(ctx, utils.GetSpokePath(retailerID), common.Name, spoke.Name)
	if err != nil {
		logger.Errorf("Error occurred while checking existence of site in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if existsSpokeName {
		logger.Debugf("Spoke with name %s already exists", spoke.Name)
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusBadRequest,
				fmt.Sprintf("Spoke with name : %s already exists", spoke.Name), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	var googleTimeZone *commonModels.GoogleTimeZone
	googleTimeZone, err = utils.GetTimeZone(ctx, *spoke.Location.Latitude, *spoke.Location.Longitude)

	if err != nil {
		logger.Errorf("Error occurred while retrieving timezone : %v", err)
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusBadRequest,
				fmt.Sprintf("Error occurred while retrieving location with latitude %f and longitude %f. "+
					"Timezone API returned with status: %s. "+
					"Please provide valid location details", *spoke.Location.Latitude,
					*spoke.Location.Longitude, googleTimeZone.Status), nil),
			response.GetCommonResponseHeaders(request))

		return
	}
	if googleTimeZone != nil && googleTimeZone.TimezoneID != "" {
		spoke.Timezone = googleTimeZone.TimezoneID
	}
	var updateTime time.Time
	var retryCount int
	spoke, updateTime, retryCount, err = performPostSpoke(responseWriter,
		request, spoke, siteSpoke, retailerID, siteID, dbClient)
	if err != nil {
		return
	}

	if retryCount == common.MaxRetryCount {
		logger.Errorf("Unable to create spoke after %d retries "+
			"as function was unable to generate unique id : %v", common.MaxRetryCount, err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	sendPostResponse(ctx, responseWriter, request, pubsubClient, spoke, siteSpoke, updateTime)
}

func performPostSpoke(responseWriter http.ResponseWriter, request *http.Request,
	spoke models.Spoke, siteSpoke models.SiteSpoke, retailerID string,
	siteID string, dbClient cloud.DB) (models.Spoke, time.Time, int, error) {
	ctx := request.Context()
	logger := logging.GetLoggerFromContext(ctx)
	var retryCount int
	var updateTime time.Time

	for retryCount = 0; retryCount < common.MaxRetryCount; retryCount++ {
		logger.Debugf("Creating unique ID for Spoke for %d time", retryCount)
		spoke.ID = fmt.Sprintf("%s%s", common.SpokeIDPrefix, utils.GetRandomID(common.RandomIDLength))

		spoke.RetailerID = retailerID

		spoke.CreatedBy = common.User
		spoke.UpdatedBy = common.User

		currentTime := time.Now().UTC().Round(time.Second)
		spoke.CreatedTime = &currentTime
		spoke.UpdatedTime = &currentTime

		siteSpoke = models.NewSiteSpoke(siteID, spoke.ID, retailerID, common.User)

		idExists, err := dbClient.ExistsInCollectionGroup(ctx, common.SpokesCollection, common.ID, spoke.ID)
		if err != nil {
			logger.Errorf("Error occurred while checking existence of spoke in DB: %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return spoke, time.Time{}, retryCount, err
		}
		if !idExists {
			logger.Debugf("Proceed with save")
			_, err = dbClient.Save(ctx, utils.GetSpokePath(retailerID), spoke.ID, spoke)
			if err != nil {
				logger.Errorf("Unable to create spoke : %v", err)
				response.RespondWithInternalServerError(responseWriter, request)

				return spoke, time.Time{}, retryCount, err
			}
			logger.Debugf("Created unique ID for site")

			logger.Debugf("Proceed with site-spoke association")
			updateTime, err = dbClient.Save(ctx, utils.GetSiteSpokePath(retailerID), siteSpoke.ID, siteSpoke)
			if err != nil {
				logger.Errorf("Unable to create site spoke association : %v", err)
				response.RespondWithInternalServerError(responseWriter, request)

				return spoke, time.Time{}, retryCount, err
			}

			break
		}
	}

	return spoke, updateTime, retryCount, nil
}

func sendPostResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	pubsubClient cloud.Queue, Spoke models.Spoke, siteSpoke models.SiteSpoke, updateTime time.Time) {
	logger := logging.GetLoggerFromContext(ctx)
	etag, err := utils.GetETag(Spoke)
	if err != nil {
		logger.Errorf("Error while getting etag for spoke struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.Respond(responseWriter, http.StatusCreated, Spoke,
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)).
			WithHeader(common.HeaderLocation, fmt.Sprintf("%s%s", common.SpokePath, Spoke.ID)).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Spoke successfully created with id : %s", Spoke.ID)

	pubsubClient.Publish(ctx, os.Getenv(common.EnvSpokeMessageTopic),
		models.GetPubSubSpokeMessage(Spoke.RetailerID, siteSpoke.SiteID, Spoke.ID,
			siteSpoke.ID, common.ChangeTypeCreate))
}
