package sites

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	siteCommon "github.com/TakeoffTech/site-info-svc/cloud-functions/sites/common"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/dbutil"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	commonModel "github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/fatih/structs"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"net/http"
	"os"
	"reflect"
	"time"
)

// This file has the function and handler to update a site into the DB
var patchSitePath = urit.MustCreateTemplate(fmt.Sprintf("/sites/{%s}", common.PathParamSiteID))

func init() {
	functions.HTTP("PatchSite", patchSite)
}

func patchSite(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("patch_site.patchSite"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	patchSiteHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()),
		cloud.NewPubSubRepository(requestWithContext.Context()))
}

func patchSiteHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("patch_site.patchSiteHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	var site models.Site

	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(models.GetRequiredHeaders(), common.HeaderIfMatch),
		RequiredPath:    patchSitePath,
		RequestMethod:   http.MethodPatch,
		RequestBodyValidation: &utils.RequestBodyValidation{
			Entity:             &site,
			CompleteValidation: false,
		},
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}
	//Gets Retailer ID from header Params
	retailerID := request.Header.Get(common.HeaderRetailerID)
	logger.Debugf("retailer id is %v", retailerID)
	//Gets Site ID from Query Params
	siteID := pathParams[common.PathParamSiteID]

	oldSiteDataMap, err := validateRequestData(ctx, responseWriter, request, retailerID, siteID, dbClient)
	if err != nil {
		return
	}

	var oldSiteData models.Site
	err = utils.ConvertToObject(oldSiteDataMap, &oldSiteData)
	if err != nil {
		logger.Errorf("Error while unmarshalling data from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	newSiteData, err := checkNameForUpdate(ctx, responseWriter, request, site, oldSiteData, retailerID, dbClient)
	if err != nil {
		return
	}

	isTimezoneChanged := false
	// Location can't be 0,0 or empty.

	newSiteData, isTimezoneChanged, err = checkLocationForUpdate(ctx, responseWriter, request, site, newSiteData)
	if err != nil {
		return
	}

	if !reflect.DeepEqual(newSiteData, oldSiteData) {
		updatedTime := time.Now().UTC().Round(time.Second)
		newSiteData.UpdatedBy = common.User
		newSiteData.UpdatedTime = &updatedTime
		docForUpdate := createDocForUpdate(newSiteData, oldSiteData)
		updateTime, err := dbClient.Update(ctx, utils.GetSitePath(newSiteData.RetailerID), newSiteData.ID, docForUpdate)
		if err != nil {
			logger.Errorf("Error while deleting site from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}

		sendPatchResponse(ctx, responseWriter, request, newSiteData, updateTime, isTimezoneChanged)
		pubsubClient.Publish(ctx, os.Getenv(common.EnvAuditLogTopic),
			audit.GetPubSubAuditMessage(audit.GetSiteAuditPath(newSiteData.RetailerID, newSiteData.ID),
				request.Header.Get(common.HeaderXCorrelationID), newSiteData.UpdatedBy,
				common.AuditTypeUpdate,
				common.EntitySite,
				newSiteData.UpdatedTime,
				structs.Map(oldSiteData),
				structs.Map(newSiteData),
			))

		pubsubClient.Publish(ctx, os.Getenv(common.EnvSiteMessageTopic),
			models.GetPubSubSiteMessage(newSiteData.RetailerID, newSiteData.ID, common.ChangeTypeUpdate))
	} else {
		// valid json and site object is same as in db.
		logger.Errorf("site object not changed \nold object: %v \nnew object: %v", oldSiteData, newSiteData)
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusUnprocessableEntity,
				"No changes detected", nil),
			response.GetCommonResponseHeaders(request))
	}
}

func checkGoogleTimeZone(googleTimeZone *commonModel.GoogleTimeZone) bool {
	if googleTimeZone != nil && googleTimeZone.TimezoneID != "" {
		return true
	}

	return false
}

func checkSiteLocation(site, newSiteData models.Site) bool {
	if site.IsValidLocationData() && !reflect.DeepEqual(newSiteData.Location, site.Location) {
		return true
	}

	return false
}

func validateRequestData(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	retailerID, siteID string, dbClient cloud.DB) (map[string]interface{}, error) {
	logger := logging.GetLoggerFromContext(ctx)
	if !dbutil.IsRetailerIDPresentInDB(responseWriter, request, dbClient, retailerID, logger, true) {
		return nil, errors.New("retailer not found")
	}

	oldSiteDataMap := siteCommon.GetSiteFromDB(responseWriter, request, logger, dbClient, retailerID, siteID, true)
	if oldSiteDataMap == nil {
		return nil, errors.New("site not found")
	}

	if !utils.IsValidEtagPresentInHeader(responseWriter, request, oldSiteDataMap, logger) {
		return nil, errors.New("invalid etag")
	}

	return oldSiteDataMap, nil
}

func checkLocationForUpdate(ctx context.Context, responseWriter http.ResponseWriter,
	request *http.Request, site, newSiteData models.Site) (models.Site, bool, error) {
	newSite := newSiteData
	isTimezoneChanged := false
	if checkSiteLocation(site, newSite) {
		newSite.Location = site.Location
		googleTimeZone, err := utils.GetTimeZone(ctx, *site.Location.Latitude, *site.Location.Longitude)
		if err != nil {
			logging.GetLoggerFromContext(ctx).Errorf("Error occurred while retrieving timezone : %v", err)
			response.RespondWithResponseObject(responseWriter,
				response.NewResponse(http.StatusBadRequest,
					fmt.Sprintf("Error occurred while retrieving location with latitude %f and longitude %f. "+
						"Timezone API returned with status: %s. "+
						"Please provide valid location details", *site.Location.Latitude, *site.Location.Longitude,
						googleTimeZone.Status), nil),
				response.GetCommonResponseHeaders(request))

			return newSite, false, err
		}
		if checkGoogleTimeZone(googleTimeZone) {
			newSite.Timezone = googleTimeZone.TimezoneID
			isTimezoneChanged = true
		}
	}

	return newSite, isTimezoneChanged, nil
}

func checkNameForUpdate(ctx context.Context, responseWriter http.ResponseWriter,
	request *http.Request, site, oldSiteData models.Site, retailerID string, dbClient cloud.DB) (models.Site, error) {
	logger := logging.GetLoggerFromContext(ctx)
	newSiteData := oldSiteData
	if site.Name != "" {
		//Check if Site name already exists
		siteNameExists, err := dbClient.Exists(ctx, utils.GetSitePath(retailerID), common.Name, site.Name)
		if err != nil {
			logger.Errorf("Error occurred while checking existence of site in DB: %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return newSiteData, errors.New("error occurred while checking existence of site in DB")
		}
		if siteNameExists {
			logger.Debugf("Site with same name already exists")
			response.RespondWithResponseObject(responseWriter,
				response.NewResponse(http.StatusUnprocessableEntity,
					fmt.Sprintf("Site with name : %s already exists", site.Name), nil),
				response.GetCommonResponseHeaders(request))

			return newSiteData, errors.New("site with name already exists")
		}
		newSiteData.Name = site.Name
	} else {
		// Name is not changed here.
		// Assign old name back to new object so later in comparison will help.
		newSiteData.Name = oldSiteData.Name
	}

	return newSiteData, nil
}

func createDocForUpdate(site models.Site, oldData models.Site) []firestore.Update {
	var docForUpdate []firestore.Update
	if site.Name != oldData.Name {
		docForUpdate = append(docForUpdate, firestore.Update{Path: "name", Value: site.Name})
	}
	if site.Location != nil {
		docForUpdate = append(docForUpdate, firestore.Update{Path: "location", Value: site.Location})
		docForUpdate = append(docForUpdate, firestore.Update{Path: "timezone", Value: site.Timezone})
	}
	docForUpdate = append(docForUpdate, firestore.Update{Path: "updated_time", Value: site.UpdatedTime})
	docForUpdate = append(docForUpdate, firestore.Update{Path: "updated_by", Value: site.UpdatedBy})

	return docForUpdate
}

func sendPatchResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	siteData models.Site, updateTime time.Time, isTimezoneChanged bool) {
	logger := logging.GetLoggerFromContext(ctx)
	etag, err := utils.GetETag(siteData)
	if err != nil {
		logger.Errorf("Error while getting etag for retailer struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	responseHeaders := response.GetCommonResponseHeaders(request)
	if isTimezoneChanged {
		responseHeaders.WithHeader(common.HeaderTimezone, siteData.Timezone)
	}
	response.Respond(responseWriter, http.StatusOK, siteData,
		responseHeaders.WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Site ID : %s updated successfully.", siteData.ID)
}
