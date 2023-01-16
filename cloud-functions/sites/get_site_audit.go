package sites

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	auditModels "github.com/TakeoffTech/site-info-svc/common/audit/models"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"net/http"
	"time"
)

// This file has the function and handler to get audit logs for a site from DB
var getSiteAuditPath = urit.MustCreateTemplate(fmt.Sprintf("/sites/{%s}/auditLogs", common.PathParamSiteID))

func init() {
	functions.HTTP("GetSiteAudit", getSiteAudit)
}

func getSiteAudit(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_site_audit.getSiteAudit"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getSiteAuditHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getSiteAuditHandler(responseWriter http.ResponseWriter,
	request *http.Request, dbClient cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_site_audit.getSiteAuditHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(models.GetRequiredHeaders(), utils.AddPaginationHeaderIfNotAdded(request)...),
		RequiredPath:    getSiteAuditPath,
		RequestMethod:   http.MethodGet,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	siteID := pathParams[common.PathParamSiteID]
	retailerID := request.Header.Get(common.HeaderRetailerID)

	exists, err := dbClient.Exists(ctx, utils.GetSitePath(retailerID), common.ID, siteID)
	if err != nil {
		logger.Errorf("Error occurred while checking existence of site for a retailer in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if !exists {
		logger.Debugf("Site with id %s under Retailer with id %s does not exist", siteID, retailerID)
		response.RespondWithNotFoundErrorMessage(responseWriter, request,
			fmt.Sprintf("Site with id %s does not exist for Retailer with id %s", siteID, retailerID), err)

		return
	}

	var data []map[string]interface{}
	var startAfterID, nextPageToken string
	pageSize := utils.GetPageSizeFromHeader(request, logger)

	if request.Header.Get(common.HeaderPageToken) != "" {
		startAfterID, err = utils.DecodeNextPageToken(request.Header.Get(common.HeaderPageToken),
			common.SitesEncryptionKey)
		logger.Debugf("ID got after decoding the next page token : %s", startAfterID)
		if err != nil {
			logger.Errorf("Error occurred while decoding the next page token : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}
	}
	parsedTime, _ := time.Parse(common.TimeParseFormat, startAfterID)
	data, startAfterID, err = dbClient.GetAll(ctx, audit.GetSiteAuditPath(retailerID, siteID),
		cloud.Page{
			StartAfterID: parsedTime,
			PageSize:     pageSize,
			OrderBy:      common.ChangedAt,
			Sort:         common.SortDescending,
		}, nil)

	if err != nil {
		logger.Errorf("Internal server error while fetching the site audit logs from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	if startAfterID != "" && len(data) == pageSize {
		logger.Debugf("Received startAfterID : %s", startAfterID)
		nextPageToken, err = utils.GetNextPageToken(startAfterID, common.SitesEncryptionKey)
		logger.Debugf("Received nextPageToken : %s", nextPageToken)
		if err != nil {
			logger.Errorf("Error occurred while creating the next page token : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}
	}
	auditLog := &auditModels.AuditLog{}
	utils.CreateResponseForGetAllByModel(ctx, responseWriter, request, data, nextPageToken, auditLog)
}
