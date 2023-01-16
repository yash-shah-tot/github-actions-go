package retailers

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/TakeoffTech/go-telemetry/sdpropagation"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/retailers/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/fatih/structs"
	"github.com/go-andiamo/urit"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"os"
	"time"
)

// This file has the function and handler to deactivate a retailer from the DB

var PostRetailerDeactivatePath = urit.MustCreateTemplate(fmt.Sprintf("/retailers/{%s}:%s",
	common.PathParamRetailerID, common.PathParamDeactivate))

func init() {
	functions.HTTP("PostRetailerDeactivate", postRetailerDeactivate)
}

func postRetailerDeactivate(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("deactivate_retailer.postRetailerDeactivate"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	postRetailerDeactivateHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()), cloud.NewPubSubRepository(requestWithContext.Context()))
}

func postRetailerDeactivateHandler(responseWriter http.ResponseWriter, request *http.Request,
	firestoreClient cloud.DB, pubSubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(),
		utils.GetSpanName("deactivate_retailer.postRetailerDeactivateHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)

	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(common.GetMandatoryHeaders(), common.HeaderIfMatch),
		RequiredPath:    PostRetailerDeactivatePath,
		RequestMethod:   http.MethodPost,
	})

	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}
	retailerID := pathParams[common.PathParamRetailerID]
	data, err := firestoreClient.GetByID(ctx, common.RetailersCollection, retailerID, true)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Retailer ID %s does not exist", retailerID), err)
		} else {
			logger.Errorf("Error occurred while checking existence of retailer in DB while deletion : %v", err)
			response.RespondWithInternalServerError(responseWriter, request)
		}

		return
	}

	//Check if all sites in retailer are 'deprecated'
	noActiveSites, err := firestoreClient.CheckSubDocuments(ctx, utils.GetSitePath(retailerID), retailerID)
	if err != nil {
		logger.Errorf("Error while fetching the retailer from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if !noActiveSites {
		logger.Debugf("deactivate request cannot be processed, there are active sites under the said retailer.")
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusPreconditionFailed,
				"Deactivate request cannot be processed, there are active sites under the said retailer.", nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	if !utils.IsValidEtagPresentInHeader(responseWriter, request, data, logger) {
		return
	}

	var retailer models.Retailer
	err = utils.ConvertToObject(data, &retailer)
	if err != nil {
		logger.Errorf("Error while converting data got from DB to retailer struct : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	deactivatedTime := time.Now().UTC().Round(time.Second)
	oldRetailer := retailer
	retailer.DeactivatedTime = &deactivatedTime
	retailer.UpdatedTime = &deactivatedTime
	retailer.UpdatedBy = common.User
	retailer.DeactivatedBy = common.User

	updatesForDelete := createUpdatesForDelete(retailer)
	updateTime, err := firestoreClient.Update(ctx, common.RetailersCollection, retailerID, updatesForDelete)
	if err != nil {
		logger.Errorf("Error while deleting retailer from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	etag, err := utils.GetETag(retailer)
	if err != nil {
		logger.Errorf("Error while getting etag for retailer struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.RespondWithResponseObject(responseWriter,
		response.NewResponse(http.StatusOK,
			fmt.Sprintf("Retailer %s deactivated successfully", retailerID), nil),
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Retailer id %s successfully deactivated", retailerID)

	pubSubClient.Publish(ctx, os.Getenv(common.EnvAuditLogTopic),
		audit.GetPubSubAuditMessage(audit.GetRetailerAuditPath(oldRetailer.ID),
			request.Header.Get(common.HeaderXCorrelationID), retailer.DeactivatedBy,
			common.AuditTypeDeactivate,
			common.EntityRetailer,
			retailer.DeactivatedTime,
			structs.Map(oldRetailer),
			nil,
		))

	pubSubClient.Publish(ctx, os.Getenv(common.EnvRetailerMessageTopic),
		models.GetPubSubRetailerMessage(retailer.ID, common.AuditTypeDeactivate))
}

func createUpdatesForDelete(retailer models.Retailer) []firestore.Update {
	var updatesForDelete []firestore.Update

	updatesForDelete = append(updatesForDelete,
		firestore.Update{Path: "deactivated_time", Value: retailer.DeactivatedTime})
	updatesForDelete = append(updatesForDelete, firestore.Update{Path: "updated_time", Value: retailer.UpdatedTime})
	updatesForDelete = append(updatesForDelete, firestore.Update{Path: "deactivated_by", Value: retailer.DeactivatedBy})
	updatesForDelete = append(updatesForDelete, firestore.Update{Path: "updated_by", Value: retailer.UpdatedBy})

	return updatesForDelete
}
