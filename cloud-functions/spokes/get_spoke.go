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
	"strings"
)

var getSpokePath = urit.MustCreateTemplate(fmt.Sprintf("/spokes/{%s}", common.PathParamSpokeID))

func init() {
	functions.HTTP("GetSpoke", getSpoke)
}

func getSpoke(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_spoke.getSpoke"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getSpokeHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getSpokeHandler(responseWriter http.ResponseWriter,
	request *http.Request, dbClient cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_spoke.getSpokeHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: models.GetRequiredHeaders(),
		RequiredPath:    getSpokePath,
		RequestMethod:   http.MethodGet,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	spokeID := pathParams[common.PathParamSpokeID]
	//Check if retailer exists
	retailerID := request.Header.Get(common.HeaderRetailerID)
	//Set delete flag
	skipDeactivated := true

	if !dbutil.IsRetailerIDPresentInDB(responseWriter, request, dbClient, retailerID, logger, skipDeactivated) {
		return
	}

	if strings.ToLower(request.URL.Query().Get(common.QueryParamDeactivated)) == common.True {
		skipDeactivated = false
	}

	data := spokesCommon.GetSpokeFromDB(responseWriter, request, logger, dbClient, retailerID, spokeID, skipDeactivated)
	if data == nil {
		return
	}

	etag, err := utils.GetETag(data)
	if err != nil {
		logger.Errorf("Error while getting etag for spoke data from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	var spoke models.Spoke
	err = utils.ConvertToObject(data, &spoke)
	if err != nil {
		logger.Errorf("Error while converting data from DB to struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.Respond(responseWriter, http.StatusOK,
		spoke,
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Spoke id %s successfully fetched for retailer : %v", spokeID, spoke)
}
