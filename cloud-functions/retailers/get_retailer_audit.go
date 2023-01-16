package retailers

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	"github.com/TakeoffTech/site-info-svc/common/audit/models"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"net/http"
	"time"
)

// This file has the function and handler to get audit logs for a retailer from DB
var getRetailerAuditPath = urit.MustCreateTemplate(fmt.Sprintf("/retailers/{%s}/auditLogs", common.PathParamRetailerID))

func init() {
	functions.HTTP("GetRetailerAudit", getRetailerAudit)
}

func getRetailerAudit(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_retailer_audit.getRetailerAudit"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getRetailerAuditHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getRetailerAuditHandler(responseWriter http.ResponseWriter,
	request *http.Request, dbClient cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_retailer_audit.getRetailerAuditHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(common.GetMandatoryHeaders(), utils.AddPaginationHeaderIfNotAdded(request)...),
		RequiredPath:    getRetailerAuditPath,
		RequestMethod:   http.MethodGet,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	retailerID := pathParams[common.PathParamRetailerID]
	exists, err := dbClient.Exists(ctx, common.RetailersCollection, common.ID, retailerID)
	if err != nil {
		logger.Errorf("Error occurred while checking existence of retailer in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if !exists {
		response.RespondWithNotFoundErrorMessage(responseWriter, request,
			fmt.Sprintf("Retailer with id : %s does not exist", retailerID), err)

		return
	}

	var data []map[string]interface{}
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
	parsedTime, _ := time.Parse(common.TimeParseFormat, startAfterID)

	data, startAfterID, err = dbClient.GetAll(ctx, audit.GetRetailerAuditPath(retailerID),
		cloud.Page{
			StartAfterID: parsedTime,
			PageSize:     pageSize,
			OrderBy:      common.ChangedAt,
			Sort:         common.SortDescending,
		}, nil)

	if err != nil {
		logger.Errorf("Internal server error while fetching the retailer audit logs from DB : %v", err)
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

	auditLog := &models.AuditLog{}
	utils.CreateResponseForGetAllByModel(ctx, responseWriter, request, data, nextPageToken, auditLog)
}
