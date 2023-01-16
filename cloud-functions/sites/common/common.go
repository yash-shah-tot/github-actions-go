package common

import (
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

func GetSiteFromDB(responseWriter http.ResponseWriter, request *http.Request, logger *zap.SugaredLogger,
	dbClient cloud.DB, retailerID string, siteID string, skipDeactivated bool) map[string]interface{} {
	// get the site from the db.
	oldSiteDataMap, err := dbClient.GetByID(request.Context(), utils.GetSitePath(retailerID), siteID, skipDeactivated)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Site ID %s not found", siteID), err)
		} else {
			logger.Errorf("Internal server error while fetching the site from DB : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return nil
	}

	return oldSiteDataMap
}
