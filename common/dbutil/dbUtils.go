package dbutil

import (
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

// IsRetailerIDPresentInDB will check retailer with given retailer ID present in db.
func IsRetailerIDPresentInDB(responseWriter http.ResponseWriter, request *http.Request, dbClient cloud.DB,
	retailerID string, logger *zap.SugaredLogger, skipDeactivated bool) bool {
	//Checks retailer if ID exists in DB
	_, err := dbClient.GetByID(request.Context(), common.RetailersCollection, retailerID, skipDeactivated)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Retailer ID %s not found", retailerID), err)
		} else {
			logger.Errorf("Internal server error while fetching the retailer from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return false
	}

	return true
}

func IsSiteIDPresentInDB(responseWriter http.ResponseWriter, request *http.Request, dbClient cloud.DB,
	retailerID string, siteID string, logger *zap.SugaredLogger, skipDeactivated bool) bool {
	//Checks site if ID exists in DB
	_, err := dbClient.GetByID(request.Context(), utils.GetSitePath(retailerID), siteID, skipDeactivated)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Site ID %s not found", siteID), err)
		} else {
			logger.Errorf("Internal server error while fetching the site from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return false
	}

	return true
}

func IsSpokeIDPresentInDB(responseWriter http.ResponseWriter, request *http.Request, dbClient cloud.DB,
	retailerID string, spokeID string, logger *zap.SugaredLogger, skipDeactivated bool) bool {
	//Checks spoke if ID exists in DB
	_, err := dbClient.GetByID(request.Context(), utils.GetSpokePath(retailerID), spokeID, skipDeactivated)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Spoke ID %s not found", spokeID), err)
		} else {
			logger.Errorf("Internal server error while fetching the site from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return false
	}

	return true
}
