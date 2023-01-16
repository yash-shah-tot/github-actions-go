package spokes

import (
	"context"
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
	"strings"
)

// This file has the function and handler to get list spokes from the DB
var getSpokesPath = urit.MustCreateTemplate("/spokes")

func init() {
	functions.HTTP("GetSpokes", getSpokes)
}

func getSpokes(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_spokes.getSpokes"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getSpokesHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getSpokesHandler(responseWriter http.ResponseWriter,
	request *http.Request, dbClient cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_spokes.getSpokesHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	_, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(models.GetRequiredHeaders(), utils.AddPaginationHeaderIfNotAdded(request)...),
		RequiredPath:    getSpokesPath,
		RequestMethod:   http.MethodGet,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}
	//Gets Retailer ID from header Params
	retailerID := request.Header.Get(common.HeaderRetailerID)
	//By default, deleted records are omitted from response.
	skipDeactivated := true

	if !dbutil.IsRetailerIDPresentInDB(responseWriter, request, dbClient, retailerID, logger, skipDeactivated) {
		return
	}

	var err error
	var nextPageToken, startAfterID string
	pageSize := utils.GetPageSizeFromHeader(request, logger)

	if request.Header.Get(common.HeaderPageToken) != "" {
		startAfterID, err = utils.DecodeNextPageToken(request.Header.Get(common.HeaderPageToken),
			common.SpokesEncryptionKey)
		logger.Debugf("ID got after decoding the next page token : %s", startAfterID)
		if err != nil {
			logger.Errorf("Error occurred while decoding the next page token : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}
	}

	var data []map[string]interface{}
	where := []cloud.Where{{
		Field:    common.DeactivatedTime,
		Operator: common.OperatorEquals,
		Value:    nil,
	}}
	if strings.ToLower(request.URL.Query().Get(common.QueryParamDeactivated)) == common.True {
		where = nil
	}
	data, startAfterID, err = dbClient.GetAll(ctx, utils.GetSpokePath(retailerID),
		cloud.Page{
			StartAfterID: startAfterID,
			PageSize:     pageSize,
			OrderBy:      common.ID,
			Sort:         common.SortAscending,
		}, where)

	if err != nil {
		logger.Errorf("Internal server error while fetching the spokes from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	if startAfterID != "" && len(data) == pageSize {
		logger.Debugf("Received startAfterID : %s", startAfterID)
		nextPageToken, err = utils.GetNextPageToken(startAfterID, common.SpokesEncryptionKey)
		logger.Debugf("Received nextPageToken : %s", nextPageToken)
		if err != nil {
			logger.Errorf("Error occurred while creating the next page token : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}
	}

	spoke := models.Spoke{}
	utils.CreateResponseForGetAllByModel(ctx, responseWriter, request, data, nextPageToken, spoke)
}
