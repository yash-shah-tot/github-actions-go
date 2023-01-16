package sites

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
	model "github.com/TakeoffTech/site-info-svc/cloud-functions/spokes/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/dbutil"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"math"
	"net/http"
	"strings"
)

var getSiteSpokesPath = urit.MustCreateTemplate(fmt.Sprintf("/sites/{%s}/spokes", common.PathParamSiteID))

func init() {
	functions.HTTP("GetSiteSpokes", getSiteSpokes)
}

func getSiteSpokes(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("get_site_spokes.getSiteSpokes"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	getSiteSpokesHandler(responseWriter, requestWithContext, cloud.NewFirestoreRepository(requestWithContext.Context()))
}

func getSiteSpokesHandler(responseWriter http.ResponseWriter,
	request *http.Request, dbClient cloud.DB) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("get_site_spokes.getSiteSpokesHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(models.GetRequiredHeaders(), utils.AddPaginationHeaderIfNotAdded(request)...),
		RequiredPath:    getSiteSpokesPath,
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
	//Set deactivate flag
	skipDeactivated := true

	if !dbutil.IsRetailerIDPresentInDB(responseWriter, request, dbClient, retailerID, logger, skipDeactivated) {
		return
	}

	if !dbutil.IsSiteIDPresentInDB(responseWriter, request, dbClient, retailerID, siteID, logger, skipDeactivated) {
		return
	}

	//Get Spoke IDs from collection
	siteSpokeData, _, err := dbClient.GetAll(ctx, utils.GetSiteSpokePath(retailerID),
		cloud.Page{
			StartAfterID: "",
			PageSize:     math.MaxInt,
			OrderBy:      common.ID,
			Sort:         common.SortAscending,
		}, []cloud.Where{{
			Field:    common.SiteID,
			Operator: common.OperatorEquals,
			Value:    siteID,
		}})

	if err != nil {
		logger.Errorf("Internal server error while fetching the site spokes mapping from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	var siteSpokeMappingData []model.SiteSpoke
	err = utils.ConvertToObject(siteSpokeData, &siteSpokeMappingData)
	if err != nil {
		logger.Errorf("Internal server error while converting Site Spoke Mpping data to struct : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	var spokeIDs []string
	for _, siteSpokeMapping := range siteSpokeMappingData {
		spokeIDs = append(spokeIDs, siteSpokeMapping.SpokeID)
	}

	var startAfterID, nextPageToken string
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

	if len(spokeIDs) == 0 {
		utils.CreateResponseForGetAllByModel(ctx, responseWriter, request, data, "", model.Spoke{})

		return
	}
	where := populateWhereClause(request, spokeIDs)

	data, startAfterID, err = dbClient.GetAll(ctx, utils.GetSpokePath(retailerID),
		cloud.Page{
			StartAfterID: startAfterID,
			PageSize:     pageSize,
			OrderBy:      common.Name,
			Sort:         common.SortAscending,
		}, where)

	if err != nil {
		logger.Errorf("Internal server error while fetching the sites from DB : %v", err)
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

	spoke := model.Spoke{}
	utils.CreateResponseForGetAllByModel(ctx, responseWriter, request, data, nextPageToken, spoke)
}

func populateWhereClause(request *http.Request, spokeIDs []string) []cloud.Where {
	where := []cloud.Where{{
		Field:    common.DeactivatedTime,
		Operator: common.OperatorEquals,
		Value:    nil,
	}}
	if strings.ToLower(request.URL.Query().Get(common.QueryParamDeactivated)) == common.True {
		where = nil
	}

	where = append(where, cloud.Where{
		Field:    common.ID,
		Operator: common.OperatorIn,
		Value:    spokeIDs,
	})

	return where
}
