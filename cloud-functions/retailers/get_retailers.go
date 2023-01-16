package retailers

import (
	"context"
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
	"net/http"
	"strings"
)

// This file has the function and handler to get list retailers from the DB
var getRetailersPath = urit.MustCreateTemplate("/retailers")

func init() {
	functions.HTTP("GetRetailers", getRetailers)
}

func getRetailers(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_retailers.getRetailers"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getRetailersHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getRetailersHandler(responseWriter http.ResponseWriter,
	request *http.Request, client cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_retailers.getRetailersHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)

	_, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(common.GetMandatoryHeaders(), utils.AddPaginationHeaderIfNotAdded(request)...),
		RequiredPath:    getRetailersPath,
		RequestMethod:   http.MethodGet,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	var data []map[string]interface{}
	var err error
	var startAfterID, nextPageToken string
	pageSize := utils.GetPageSizeFromHeader(request, logger)

	if request.Header.Get(common.HeaderPageToken) != "" {
		startAfterID, err = utils.DecodeNextPageToken(request.Header.Get(common.HeaderPageToken),
			common.RetailersEncryptionKey)
		logger.Debugf("ID got after decoding the next page token : %s", startAfterID)
		if err != nil {
			logger.Errorf("Error occurred while decoding the next page token : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}
	}

	where := []cloud.Where{{
		Field:    common.DeactivatedTime,
		Operator: common.OperatorEquals,
		Value:    nil,
	}}
	if strings.ToLower(request.URL.Query().Get(common.QueryParamDeactivated)) == common.True {
		where = nil
	}

	data, startAfterID, err = client.GetAll(ctx, common.RetailersCollection,
		cloud.Page{
			StartAfterID: startAfterID,
			PageSize:     pageSize,
			OrderBy:      common.ID,
			Sort:         common.SortAscending,
		}, where)

	if err != nil {
		logger.Errorf("Internal server error while fetching the retailers from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	if startAfterID != "" && len(data) == pageSize {
		logger.Debugf("Received startAfterID : %s", startAfterID)
		nextPageToken, err = utils.GetNextPageToken(startAfterID, common.RetailersEncryptionKey)
		logger.Debugf("Received nextPageToken : %s", nextPageToken)
		if err != nil {
			logger.Errorf("Error occurred while creating the next page token : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}
	}
	retailer := &models.Retailer{}
	utils.CreateResponseForGetAllByModel(ctx, responseWriter, request, data, nextPageToken, retailer)
}
