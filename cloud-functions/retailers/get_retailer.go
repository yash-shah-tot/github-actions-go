package retailers

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/retailers/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strings"
)

// This file has the function and handler to get a retailer from the DB
var getRetailerPath = urit.MustCreateTemplate(fmt.Sprintf("/retailers/{%s}", common.PathParamRetailerID))

func init() {
	functions.HTTP("GetRetailer", getRetailer)
}

func getRetailer(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_retailer.getRetailer"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getRetailerHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getRetailerHandler(responseWriter http.ResponseWriter,
	request *http.Request, dbClient cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_retailer.getRetailerHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: common.GetMandatoryHeaders(),
		RequiredPath:    getRetailerPath,
		RequestMethod:   http.MethodGet,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}
	skipDeactivated := true
	if strings.ToLower(request.URL.Query().Get(common.QueryParamDeactivated)) == common.True {
		skipDeactivated = false
	}
	retailerID := pathParams[common.PathParamRetailerID]
	data, err := dbClient.GetByID(ctx, common.RetailersCollection, retailerID, skipDeactivated)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Retailer ID %s not found", retailerID), err)
		} else {
			logger.Errorf("Error while fetching the retailer from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return
	}
	etag, err := utils.GetETag(data)
	if err != nil {
		logger.Errorf("Error while getting etag for retailer data from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	var retailer models.Retailer
	err = utils.ConvertToObject(data, &retailer)
	if err != nil {
		logger.Errorf("Error while converting data from DB to struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.Respond(responseWriter, http.StatusOK,
		retailer,
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Retailer id %s successfully fetched retailer : %v", retailerID, retailer)
}
