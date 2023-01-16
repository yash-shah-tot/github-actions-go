package common

import (
	"fmt"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/spokes/models"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

func GetSpokeFromDB(responseWriter http.ResponseWriter, request *http.Request, logger *zap.SugaredLogger,
	dbClient cloud.DB, retailerID string, spokeID string, skipDeactivated bool) map[string]interface{} {
	// get the spoke from the db.
	oldSpokeDataMap, err := dbClient.GetByID(request.Context(), utils.GetSpokePath(retailerID), spokeID, skipDeactivated)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Spoke ID %s not found", spokeID), err)
		} else {
			logger.Errorf("Internal server error while fetching the spoke from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return nil
	}

	return oldSpokeDataMap
}

// IsSpokeAttachedToSite will check whether spoke is attached to site.
func IsSpokeAttachedToSite(responseWriter http.ResponseWriter, request *http.Request, dbClient cloud.DB,
	retailerID string, siteID string, spokeID string, logger *zap.SugaredLogger) bool {
	//Checks if site & spokes are attached.
	_, err := dbClient.GetByID(request.Context(), utils.GetSiteSpokePath(retailerID),
		models.GetSiteSpokeID(siteID, spokeID), false)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Spoke ID %s is not attached to site %s", spokeID, siteID), err)
		} else {
			logger.Errorf("Internal server error while fetching the site from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return false
	}

	return true
}
