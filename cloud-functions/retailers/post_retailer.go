package retailers

import (
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
	"net/http"
	"os"
	"time"
)

// This file has the function and handler to create a retailer into the DB
var postRetailerPath = urit.MustCreateTemplate("/retailers")

func init() {
	functions.HTTP("PostRetailer", postRetailer)
}

func postRetailer(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := sdpropagation.StartSpanWithRemoteParentFromRequest(request,
		utils.GetSpanName("post_retailer.postRetailer"))
	defer span.End()
	key, logger := logging.GetContextWithLogger(request)
	requestWithContext := request.WithContext(context.WithValue(ctx, key, logger))
	postRetailerHandler(responseWriter, requestWithContext,
		cloud.NewFirestoreRepository(requestWithContext.Context()), cloud.NewPubSubRepository(requestWithContext.Context()))
}

func postRetailerHandler(responseWriter http.ResponseWriter, request *http.Request,
	dbClient cloud.DB, pubsubClient cloud.Queue) {
	ctx, span := trace.StartSpan(request.Context(), utils.GetSpanName("post_retailer.postRetailerHandler"))
	defer span.End()
	logger := logging.GetLoggerFromContext(ctx)
	var retailer models.Retailer
	_, validationResponse := utils.ValidateRequest(request, utils.RequestValidation{
		RequiredPath:    postRetailerPath,
		RequestMethod:   http.MethodPost,
		RequiredHeaders: common.GetMandatoryHeaders(),
		RequestBodyValidation: &utils.RequestBodyValidation{
			Entity:             &retailer,
			CompleteValidation: true,
		},
	})

	if validationResponse != nil {
		logger.Debugf("Request body validation failed. validationResponse : %v", validationResponse)
		response.RespondWithResponseObject(responseWriter, validationResponse, response.GetCommonResponseHeaders(request))

		return
	}

	exists, err := dbClient.Exists(ctx, common.RetailersCollection, common.Name, retailer.Name)
	if err != nil {
		logger.Errorf("Error occurred while checking existence of retailer in DB: %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	if exists {
		logger.Debugf("Retailer with name %s already exists", retailer.Name)
		response.RespondWithResponseObject(responseWriter,
			response.NewResponse(http.StatusBadRequest,
				fmt.Sprintf("Retailer with name : %s already exists", retailer.Name), nil),
			response.GetCommonResponseHeaders(request))

		return
	}

	var updateTime time.Time
	var retryCount int
	for retryCount = 0; retryCount < common.MaxRetryCount; retryCount++ {
		logger.Debugf("Creating unique ID for retailer for %d time", retryCount)
		retailer.ID = fmt.Sprintf("%s%s", common.RetailerIDPrefix, utils.GetRandomID(common.RandomIDLength))

		retailer.CreatedBy = common.User
		retailer.UpdatedBy = common.User

		currentTime := time.Now().UTC().Round(time.Second)
		retailer.CreatedTime = &currentTime
		retailer.UpdatedTime = &currentTime

		idExists, err := dbClient.Exists(ctx, common.RetailersCollection, common.ID, retailer.ID)
		if err != nil {
			logger.Errorf("Error occurred while checking existence of retailer in DB: %v", err)
			response.RespondWithInternalServerError(responseWriter, request)

			return
		}
		if !idExists {
			logger.Debugf("Created unique ID for retailer does not exist proceed with save")
			updateTime, err = dbClient.Save(ctx, common.RetailersCollection, retailer.ID, retailer)
			if err != nil {
				logger.Errorf("Unable to create retailer : %v", err)
				response.RespondWithInternalServerError(responseWriter, request)

				return
			}

			break
		}
	}

	if retryCount == common.MaxRetryCount {
		logger.Errorf("Unable to create retailer after %d retires "+
			"as function was unable to generate unique id : %v", common.MaxRetryCount, err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	sendPostResponse(ctx, responseWriter, request, pubsubClient, retailer, updateTime)
}

func sendPostResponse(ctx context.Context, responseWriter http.ResponseWriter, request *http.Request,
	pubsubClient cloud.Queue, retailer models.Retailer, updateTime time.Time) {
	logger := logging.GetLoggerFromContext(ctx)
	etag, err := utils.GetETag(retailer)
	if err != nil {
		logger.Errorf("Error while getting etag for retailer struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}

	response.Respond(responseWriter, http.StatusCreated, retailer, response.GetCommonResponseHeaders(request).
		WithHeader(common.HeaderLastModified, updateTime.Format(time.RFC3339)).
		WithHeader(common.HeaderLocation, fmt.Sprintf("%s%s", common.RetailerPath, retailer.ID)).
		WithHeader(common.HeaderEtag, etag))

	logger.Debugf("Retailer successfully created with id : %s", retailer.ID)

	pubsubClient.Publish(ctx, os.Getenv(common.EnvAuditLogTopic),
		audit.GetPubSubAuditMessage(audit.GetRetailerAuditPath(retailer.ID),
			request.Header.Get(common.HeaderXCorrelationID), retailer.CreatedBy,
			common.AuditTypeCreate,
			common.EntityRetailer,
			retailer.CreatedTime,
			nil,
			structs.Map(retailer),
		))

	pubsubClient.Publish(ctx, os.Getenv(common.EnvRetailerMessageTopic),
		models.GetPubSubRetailerMessage(retailer.ID, common.ChangeTypeCreate))
}
