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
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"net/http"
	"os"
	"time"
)

var patchSpokeAttachPath = urit.MustCreateTemplate(fmt.Sprintf("/sites/{%s}/spokes/{%s}:attach",
	common.PathParamSiteID, common.PathParamSpokeID))

func init() {
	functions.HTTP("PatchSpokeAttach", patchSpokeAttach)
}

func patchSpokeAttach(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("patch_spoke_attach.patchSpokeAttach"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	patchSpokeAttachHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()),
		cloud.NewPubSubRepository(requestWithContext.Context()))
}

func patchSpokeAttachHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("patch_spoke_attach.patchSpokeAttachHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: models.GetRequiredHeaders(),
		RequiredPath:    patchSpokeAttachPath,
		RequestMethod:   http.MethodPatch,
	})
	if validationResponse != nil {
		logger.Errorf("Request body validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	siteID := pathParams[common.PathParamSiteID]
	spokeID := pathParams[common.PathParamSpokeID]
	retailerID := request.Header.Get(common.HeaderRetailerID)
	var err error

	if !dbutil.IsRetailerIDPresentInDB(responseWriter, request, dbClient, retailerID, logger, true) {
		return
	}

	if !dbutil.IsSiteIDPresentInDB(responseWriter, request, dbClient, retailerID, siteID, logger, true) {
		return
	}

	if !dbutil.IsSpokeIDPresentInDB(responseWriter, request, dbClient, retailerID, spokeID, logger, true) {
		return
	}

	exists, err := dbClient.Exists(ctx, utils.GetSiteSpokePath(retailerID),
		common.ID, models.GetSiteSpokeID(siteID, spokeID))
	if err != nil {
		logger.Errorf("Error occurred while checking existence of site spoke mapping in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if exists {
		response.Respond(responseWriter, http.StatusBadRequest,
			response.NewResponse(http.StatusBadRequest,
				fmt.Sprintf("Spoke %s is already attached to site %s", spokeID, siteID), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	siteSpoke := models.NewSiteSpoke(siteID, spokeID, retailerID, common.User)

	updateTime, err := dbClient.Save(ctx, utils.GetSiteSpokePath(retailerID), siteSpoke.ID, siteSpoke)
	if err != nil {
		logger.Errorf("Error while attaching site and spoke from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	sendPatchSpokeAttachResponse(ctx, responseWriter, request, pubsubClient, siteSpoke, updateTime)
}

func sendPatchSpokeAttachResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	pubsubClient cloud.Queue, siteSpoke models.SiteSpoke, updateTime time.Time) {
	logger := logging.GetLoggerFromContext(ctx)

	response.Respond(responseWriter, http.StatusOK,
		response.NewResponse(http.StatusOK,
			fmt.Sprintf("Spoke %s attached successfully to site %s", siteSpoke.SpokeID, siteSpoke.SiteID), nil),
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)))

	logger.Debugf("Spoke %s attached successfully to site %s", siteSpoke.SpokeID, siteSpoke.SiteID)

	pubsubClient.Publish(ctx, os.Getenv(common.EnvSpokeMessageTopic),
		models.GetPubSubSpokeMessage(siteSpoke.RetailerID, siteSpoke.SiteID, siteSpoke.SpokeID,
			models.GetSiteSpokeID(siteSpoke.SiteID, siteSpoke.SpokeID), common.ChangeTypeUpdate))
}
