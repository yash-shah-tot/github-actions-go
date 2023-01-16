package spokes

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	spokesCommon "github.com/TakeoffTech/site-info-svc/cloud-functions/spokes/common"
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

var patchSpokeDetachPath = urit.MustCreateTemplate(fmt.Sprintf("/sites/{%s}/spokes/{%s}:detach",
	common.PathParamSiteID, common.PathParamSpokeID))

func init() {
	functions.HTTP("PatchSpokeDetach", patchSpokeDetach)
}

func patchSpokeDetach(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("patch_spoke_detach.patchSpokeDetach"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	patchSpokeDetachHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()),
		cloud.NewPubSubRepository(requestWithContext.Context()))
}

func patchSpokeDetachHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("patch_spoke_detach.patchSpokeDetachHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: models.GetRequiredHeaders(),
		RequiredPath:    patchSpokeDetachPath,
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

	if !spokesCommon.IsSpokeAttachedToSite(responseWriter, request, dbClient, retailerID, siteID, spokeID, logger) {
		return
	}

	_, err = dbClient.Delete(ctx, utils.GetSiteSpokePath(retailerID), models.GetSiteSpokeID(siteID, spokeID))
	if err != nil {
		logger.Errorf("Error while detaching site and spoke from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	sendPatchSpokeDetachResponse(ctx, responseWriter, request, pubsubClient, spokeID, siteID, retailerID)
}

func sendPatchSpokeDetachResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	pubsubClient cloud.Queue, spokeID string, siteID string, retailerID string) {
	logger := logging.GetLoggerFromContext(ctx)

	response.Respond(responseWriter, http.StatusOK,
		response.NewResponse(http.StatusOK,
			fmt.Sprintf("Spoke %s detached successfully from site %s", spokeID, siteID), nil),
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderLastModified, time.Now().Format(time.RFC3339)))

	logger.Debugf("Spoke %s detached successfully from site %s", spokeID, siteID)

	pubsubClient.Publish(ctx, os.Getenv(common.EnvSpokeMessageTopic),
		models.GetPubSubSpokeMessage(retailerID, siteID, spokeID,
			models.GetSiteSpokeID(siteID, spokeID), common.ChangeTypeUpdate))
}
