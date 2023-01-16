package sites

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	siteCommon "github.com/TakeoffTech/site-info-svc/cloud-functions/sites/common"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
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

var getSitePath = urit.MustCreateTemplate(fmt.Sprintf("/sites/{%s}", common.PathParamSiteID))

func init() {
	functions.HTTP("GetSite", getSite)
}

func getSite(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_site.getSite"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getSiteHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getSiteHandler(responseWriter http.ResponseWriter,
	request *http.Request, dbClient cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_site.getSiteHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: models.GetRequiredHeaders(),
		RequiredPath:    getSitePath,
		RequestMethod:   http.MethodGet,
	})
	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	siteID := pathParams[common.PathParamSiteID]
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

	data := siteCommon.GetSiteFromDB(responseWriter, request, logger, dbClient, retailerID, siteID, skipDeactivated)
	if data == nil {
		return
	}

	etag, err := utils.GetETag(data)
	if err != nil {
		logger.Errorf("Error while getting etag for retailer data from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	var site models.Site
	err = utils.ConvertToObject(data, &site)
	if err != nil {
		logger.Errorf("Error while converting data from DB to struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.Respond(responseWriter, http.StatusOK,
		site,
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Site id %s successfully fetched retailer : %v", siteID, site)
}
