package sites

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/dbutil"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	commonModels "github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/fatih/structs"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"net/http"
	"os"
	"time"
)

// This file has the function and handler to create a site into the DB
var postSitePath = urit.MustCreateTemplate("/sites")

func init() {
	functions.HTTP("PostSite", postSite)
}

func postSite(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("post_site.postSite"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	postSiteHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()),
		cloud.NewPubSubRepository(requestWithContext.Context()))
}

func postSiteHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("post_site.postSiteHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	var site models.Site
	_, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: models.GetRequiredHeaders(),
		RequiredPath:    postSitePath,
		RequestMethod:   http.MethodPost,
		RequestBodyValidation: &utils.RequestBodyValidation{
			Entity:             &site,
			CompleteValidation: true,
		},
	})
	if validationResponse != nil {
		logger.Errorf("Request body validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	retailerID := request.Header.Get(common.HeaderRetailerID)
	var err error

	if !dbutil.IsRetailerIDPresentInDB(responseWriter, request, dbClient, retailerID, logger, true) {
		return
	}

	existsSiteName, err := dbClient.Exists(ctx, utils.GetSitePath(retailerID), common.Name, site.Name)
	if err != nil {
		logger.Errorf("Error occurred while checking existence of site in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if existsSiteName {
		logger.Debugf("Site with name %s already exists", site.Name)
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusBadRequest,
				fmt.Sprintf("Site with name : %s already exists", site.Name), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	existsRetailersSiteID, err := dbClient.Exists(ctx, utils.GetSitePath(retailerID),
		common.RetailersSiteID, site.RetailerSiteID)
	if err != nil {
		logger.Errorf("Error occurred while checking existence of retailer's site id in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	if existsRetailersSiteID {
		logger.Debugf("Retailer's site id %s already exists", site.RetailerSiteID)
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusBadRequest,
				fmt.Sprintf("Retailer's site id %s already exists", site.RetailerSiteID), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	var googleTimeZone *commonModels.GoogleTimeZone
	googleTimeZone, err = utils.GetTimeZone(ctx, *site.Location.Latitude, *site.Location.Longitude)

	if err != nil {
		logger.Errorf("Error occurred while retrieving timezone : %v", err)
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusBadRequest,
				fmt.Sprintf("Error occurred while retrieving location with latitude %f and longitude %f. "+
					"Timezone API returned with status: %s. "+
					"Please provide valid location details", *site.Location.Latitude,
					*site.Location.Longitude, googleTimeZone.Status), nil),
			response.GetCommonResponseHeaders(request))

		return
	}
	if googleTimeZone != nil && googleTimeZone.TimezoneID != "" {
		site.Timezone = googleTimeZone.TimezoneID
	}
	var updateTime time.Time
	var retryCount int
	site, updateTime, retryCount, err = performPostSite(ctx, responseWriter, request, site, retailerID, dbClient)
	if err != nil {
		return
	}

	if retryCount == common.MaxRetryCount {
		logger.Errorf("Unable to create site after %d retries "+
			"as function was unable to generate unique id : %v", common.MaxRetryCount, err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	sendPostResponse(ctx, responseWriter, request, pubsubClient, site, updateTime)
}

func performPostSite(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	site models.Site, retailerID string, dbClient cloud.DB) (models.Site, time.Time, int, error) {
	logger := logging.GetLoggerFromContext(ctx)
	var retryCount int
	var updateTime time.Time
	newSite := site
	for retryCount = 0; retryCount < common.MaxRetryCount; retryCount++ {
		logger.Debugf("Creating unique ID for site for %d time", retryCount)
		newSite.ID = fmt.Sprintf("%s%s", common.SiteIDPrefix, utils.GetRandomID(common.RandomIDLength))
		newSite.RetailerID = retailerID

		newSite.CreatedBy = common.User
		newSite.UpdatedBy = common.User

		currentTime := time.Now().UTC().Round(time.Second)
		newSite.CreatedTime = &currentTime
		newSite.UpdatedTime = &currentTime

		newSite.Status = common.StatusDraft

		idExists, err := dbClient.ExistsInCollectionGroup(ctx, common.SitesCollection, common.ID, newSite.ID)
		if err != nil {
			logger.Errorf("Error occurred while checking existence of site in DB: %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return newSite, time.Time{}, retryCount, err
		}
		if !idExists {
			logger.Debugf("Proceed with save")
			updateTime, err = dbClient.Save(ctx, utils.GetSitePath(retailerID), newSite.ID, newSite)
			if err != nil {
				logger.Errorf("Unable to create site : %v", err)
				response.RespondWithInternalServerError(responseWriter, request)

				return newSite, time.Time{}, retryCount, err
			}
			logger.Debugf("Created unique ID for site")

			break
		}
	}

	return newSite, updateTime, retryCount, nil
}

func sendPostResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	pubsubClient cloud.Queue, site models.Site, updateTime time.Time) {
	logger := logging.GetLoggerFromContext(ctx)
	etag, err := utils.GetETag(site)
	if err != nil {
		logger.Errorf("Error while getting etag for site struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.Respond(responseWriter, http.StatusCreated, site,
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)).
			WithHeader(common.HeaderLocation, fmt.Sprintf("%s%s", common.SitePath, site.ID)).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Site successfully created with id : %s", site.ID)

	pubsubClient.Publish(ctx, os.Getenv(common.EnvAuditLogTopic),
		audit.GetPubSubAuditMessage(audit.GetSiteAuditPath(site.RetailerID, site.ID),
			request.Header.Get(common.HeaderXCorrelationID), site.CreatedBy,
			common.AuditTypeCreate,
			common.EntitySite,
			site.CreatedTime,
			nil,
			structs.Map(site),
		))

	pubsubClient.Publish(ctx, os.Getenv(common.EnvSiteMessageTopic),
		models.GetPubSubSiteMessage(site.RetailerID, site.ID, common.ChangeTypeCreate))
}
