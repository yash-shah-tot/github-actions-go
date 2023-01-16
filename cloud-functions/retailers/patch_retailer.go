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

var patchRetailerPath = urit.MustCreateTemplate(fmt.Sprintf("/retailers/{%s}", common.PathParamRetailerID))

func init() {
	functions.HTTP("PatchRetailer", patchRetailer)
}

func patchRetailer(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("patch_retailer.patchRetailer"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	patchRetailerHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()), cloud.NewPubSubRepository(requestWithContext.Context()))
}

func patchRetailerHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("patch_retailer.patchRetailerHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)

	var retailer models.Retailer
	pathParams, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredHeaders: append(common.GetMandatoryHeaders(), common.HeaderIfMatch),
		RequiredPath:    patchRetailerPath,
		RequestMethod:   http.MethodPatch,
		RequestBodyValidation: &utils.RequestBodyValidation{
			Entity:             &retailer,
			CompleteValidation: false,
		},
	})

	if validationResponse != nil {
		logger.Debugf("Request validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	//Stores Retailer ID from Query Params
	retailerID := pathParams[common.PathParamRetailerID]

	//Checks if ID exists in DB
	oldRetailerData, err := dbClient.GetByID(ctx, common.RetailersCollection, retailerID, true)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			response.RespondWithNotFoundErrorMessage(responseWriter, request,
				fmt.Sprintf("Retailer ID %s not found", retailerID), err)

			return
		}
		logger.Errorf("Internal server error while fetching the retailer from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	if !utils.IsValidEtagPresentInHeader(responseWriter, request, oldRetailerData, logger) {
		return
	}

	var retailerData models.Retailer
	err = utils.ConvertToObject(oldRetailerData, &retailerData)
	if err != nil {
		logger.Errorf("Error while unmarshalling data from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	//Check if Retailer name already exists
	exists, err := dbClient.Exists(ctx, common.RetailersCollection, common.Name, retailer.Name)
	if err != nil {
		logger.Errorf("Error occurred while checking existence of retailer in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if exists {
		logger.Debugf("Retailer with same name already exists")
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusUnprocessableEntity,
				fmt.Sprintf("Retailer with name : %s already exists", retailer.Name), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	updatedTime := time.Now().UTC().Round(time.Second)
	retailerData.Name = retailer.Name
	retailerData.UpdatedBy = common.User
	retailerData.UpdatedTime = &updatedTime

	docForUpdate := createDocForUpdate(retailerData)
	updateTime, err := dbClient.Update(ctx, common.RetailersCollection, retailerID, docForUpdate)
	if err != nil {
		logger.Errorf("Error while deleting retailer from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	sendPatchResponse(ctx, responseWriter, request, retailerData, updateTime)

	pubsubClient.Publish(ctx, os.Getenv(common.EnvAuditLogTopic),
		audit.GetPubSubAuditMessage(audit.GetRetailerAuditPath(retailerData.ID),
			request.Header.Get(common.HeaderXCorrelationID), retailerData.UpdatedBy,
			common.AuditTypeUpdate,
			common.EntityRetailer,
			retailerData.UpdatedTime,
			oldRetailerData,
			structs.Map(retailerData),
		))

	pubsubClient.Publish(ctx, os.Getenv(common.EnvRetailerMessageTopic),
		models.GetPubSubRetailerMessage(retailer.ID, common.ChangeTypeUpdate))
}

func createDocForUpdate(retailer models.Retailer) []firestore.Update {
	var docForUpdate []firestore.Update

	docForUpdate = append(docForUpdate, firestore.Update{Path: "name", Value: retailer.Name})
	docForUpdate = append(docForUpdate, firestore.Update{Path: "updated_time", Value: retailer.UpdatedTime})
	docForUpdate = append(docForUpdate, firestore.Update{Path: "updated_by", Value: retailer.UpdatedBy})

	return docForUpdate
}

func sendPatchResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	retailerData models.Retailer, updateTime time.Time) {
	logger := logging.GetLoggerFromContext(ctx)
	etag, err := utils.GetETag(retailerData)
	if err != nil {
		logger.Errorf("Error while getting etag for retailer struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.Respond(responseWriter, http.StatusOK, retailerData,
		response.GetCommonResponseHeaders(request).
			WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)).
			WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Retailer ID : %s updated successfully.", retailerData.ID)
}
