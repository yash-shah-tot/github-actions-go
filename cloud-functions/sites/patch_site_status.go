package sites

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	siteCommon "github.com/TakeoffTech/site-info-svc/cloud-functions/sites/common"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"net/http"
	"os"
	"strings"
	"time"
)

var patchSiteStatusPath = urit.MustCreateTemplate("/sites/{site_id}:{status}")
var siteStatuses models.SiteStatuses

func init() {
	functions.HTTP("PatchSiteStatus", patchSiteStatus)
}

func patchSiteStatus(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("patch_site_status.patchSiteStatus"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	patchSiteStatusHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()),
		cloud.NewPubSubRepository(requestWithContext.Context()))
}

func patchSiteStatusHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("patch_site_status.patchSiteStatusHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(models.GetRequiredHeaders(), common.HeaderIfMatch),
		RequiredPath:    patchSiteStatusPath,
		RequestMethod:   http.MethodPatch,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.Respond(responseWriter, http.StatusBadRequest, validationResponse,
			response.GetCommonResponseHeaders(request))

		return
	}
	//Stores Retailer ID from Headers
	retailerID := request.Header.Get(common.HeaderRetailerID)
	//Stores Site ID from Path Params
	siteID := pathParams[common.PathParamSiteID]
	//Stores status from Path Params
	siteStatus := strings.ToLower(pathParams[common.Status])

	err := populateSiteStatusTransitions(ctx, dbClient)
	if err != nil {
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	statusTransitionMap := siteStatuses.StatusTransitions
	if statusTransitionMap[siteStatus] == nil {
		logger.Debugf("Invalid target status got from request : %s", siteStatus)
		response.Respond(responseWriter, http.StatusBadRequest, response.NewResponse(http.StatusBadRequest,
			fmt.Sprintf("Invalid status '%s' received in the request", siteStatus), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	//Checks if site exists in DB, if exists get the site data.
	oldSiteDataMap := siteCommon.GetSiteFromDB(responseWriter, request, logger, dbClient, retailerID, siteID, true)
	if oldSiteDataMap == nil {
		return
	}

	if !utils.IsValidEtagPresentInHeader(responseWriter, request, oldSiteDataMap, logger) {
		return
	}

	var oldSiteData models.Site
	err = utils.ConvertToObject(oldSiteDataMap, &oldSiteData)
	if err != nil {
		logger.Errorf("Error while unmarshalling data from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	targetSiteStatuses := statusTransitionMap[strings.ToLower(oldSiteData.Status)]
	if targetSiteStatuses == nil {
		logger.Errorf("Site id %s of retailer id %s is in corrupted state %s", siteID, retailerID, oldSiteData.Status)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	} else if !utils.Contains(targetSiteStatuses, siteStatus) {
		logger.Debugf("Invalid target status got from request : %s", siteStatus)
		response.Respond(responseWriter, http.StatusBadRequest, response.NewResponse(http.StatusBadRequest,
			fmt.Sprintf("Invalid status transition received in the request. "+
				"The site status cannot be changed from %s to %s status", oldSiteData.Status, siteStatus), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	newSiteData := createNewSiteData(oldSiteData, siteStatus)
	docForUpdate := createDocForStatusUpdate(newSiteData)

	updateTime, err := dbClient.Update(ctx, utils.GetSitePath(retailerID), siteID, docForUpdate)
	if err != nil {
		logger.Errorf("Error while deleting site from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	sendPatchStatusResponse(ctx, responseWriter, request, newSiteData, updateTime)
	publicToPubSubTopic(ctx, request, oldSiteData, newSiteData, pubsubClient)
}

func createNewSiteData(oldSiteData models.Site, newStatus string) models.Site {
	newSiteData := oldSiteData
	updatedTime := time.Now().UTC().Round(time.Second)
	newSiteData.UpdatedBy = common.User
	newSiteData.UpdatedTime = &updatedTime
	newSiteData.Status = newStatus
	if newStatus == common.StatusDeprecated {
		newSiteData.DeactivatedTime = &updatedTime
		newSiteData.DeactivatedBy = common.User
	}

	return newSiteData
}

func createDocForStatusUpdate(site models.Site) []firestore.Update {
	var docForUpdate []firestore.Update
	docForUpdate = append(docForUpdate, firestore.Update{Path: "status", Value: site.Status})
	docForUpdate = append(docForUpdate, firestore.Update{Path: "updated_time", Value: site.UpdatedTime})
	docForUpdate = append(docForUpdate, firestore.Update{Path: "updated_by", Value: site.UpdatedBy})
	if site.Status == common.StatusDeprecated {
		docForUpdate = append(docForUpdate, firestore.Update{Path: "deactivated_by", Value: site.DeactivatedBy})
		docForUpdate = append(docForUpdate, firestore.Update{Path: "deactivated_time", Value: site.DeactivatedTime})
	}

	return docForUpdate
}

func sendPatchStatusResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	site models.Site, updateTime time.Time) {
	logger := logging.GetLoggerFromContext(ctx)
	etag, err := utils.GetETag(site)
	if err != nil {
		logger.Errorf("Error while getting etag for retailer struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	responseHeaders := response.GetCommonResponseHeaders(request)

	response.Respond(responseWriter, http.StatusOK, site,
		responseHeaders.WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Site status : %s updated successfully.", site.Status)
}

func publicToPubSubTopic(ctx context.Context, request *http.Request,
	oldSiteData models.Site, newSiteData models.Site, pubsubClient cloud.Queue) {
	pubsubClient.Publish(ctx, os.Getenv(common.EnvAuditLogTopic),
		audit.GetPubSubAuditMessage(audit.GetSiteAuditPath(newSiteData.RetailerID, newSiteData.ID),
			request.Header.Get(common.HeaderXCorrelationID), newSiteData.UpdatedBy,
			common.AuditTypeUpdate,
			common.Status,
			newSiteData.UpdatedTime,
			map[string]interface{}{common.Status: oldSiteData.Status},
			map[string]interface{}{common.Status: newSiteData.Status},
		))

	changeType := common.ChangeTypeUpdate
	if newSiteData.Status == common.StatusDeprecated {
		changeType = common.ChangeTypeDelete
	}
	pubsubClient.Publish(ctx, os.Getenv(common.EnvSiteMessageTopic),
		models.GetPubSubSiteMessage(newSiteData.RetailerID, newSiteData.ID, changeType))
}

func populateSiteStatusTransitions(ctx context.Context, dbClient cloud.DB) error {
	currentTime := time.Now()

	if siteStatuses.ExpiresAt.IsZero() || siteStatuses.ExpiresAt.Before(currentTime) {
		statusTransitionData, err := dbClient.GetByID(ctx, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false)
		if err != nil {
			logging.GetLoggerFromContext(ctx).
				Errorf("Internal server error while fetching the site status transition map from DB : %v", err)

			return err
		}

		err = utils.ConvertToObject(statusTransitionData, &siteStatuses)
		if err != nil {
			logging.GetLoggerFromContext(ctx).
				Errorf("Error while unmarshalling data from DB : %v", err)

			return err
		}
		logging.GetLoggerFromContext(ctx).
			Info("Site statuses transition map refreshed at %v", currentTime)
		siteStatuses.ExpiresAt = currentTime.Add(common.CacheRetentionTime)
	}

	return nil
}
