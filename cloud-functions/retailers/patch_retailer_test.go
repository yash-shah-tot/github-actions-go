package retailers

import (
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
}

func Test_patchRetailer(t *testing.T) {
	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Request with all required headers ",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPatch, "/retailers/r12345", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch),
			},
		},
		{
			"Request with no headers ",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPatch, "/retailers/r12345", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchRetailer(tt.args.w, tt.args.r)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_patchRetailerHandler(t *testing.T) {
	t.Run("RetailerID not passed in path param", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, "/retailers", nil)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Request does not have the required headers : [Accept-Version X-Correlation-ID If-Match]\",\"Invalid request url path, no matching path params found in path : /retailers\"]}", string(bytes))
	})

	t.Run("Invalid Request Method", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345", "{invalid:json}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Invalid JSON body in PATCH", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{invalid:json}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"invalid character 'i' looking for beginning of object key string\"]}", string(bytes))
	})

	t.Run("Missing required fields in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"Empty JSON received, please input valid JSON in body\"]}", string(bytes))
	})

	t.Run("RetailerID does not exist", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(nil, status.Error(codes.NotFound, "Retailer ID not found"))
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("Failed while getting data from DB", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(nil, errors.New("connection timeout"))
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Retailer ID is already deleted ", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set(common.HeaderIfMatch, "1f818230eecabdefa93c35acaa7807a7df2dc1935e24d106ba5bab6368601f3a")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(nil, status.Error(codes.NotFound, "document not found"))
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("ETag Mismatch", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "retailerName",
			"created_by":       "API",
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusPreconditionFailed, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":412,\"message\":\"If-Match header value incorrect, please get the latest and try again\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		retailer := map[string]interface{}{
			"foo": make(chan int),
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling invalid retailer returned from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "6c910040c3bd9bf90b07ea06a369b0db18e112cd0d3c068c361139424f6a5363")
		retailer := map[string]interface{}{
			"id": 1234,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Retailer name already exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "323190fa75e7aed77f65eca7ae2c5a07a6c466d7bf78865ede5c57f9ae8ffca8")
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "retailerName",
			"created_by":       "API",
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", "name", "retailerName").Return(true, nil)
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":422,\"message\":\"Retailer with name : retailerName already exists\"}", string(bytes))
	})

	t.Run("DB Down during exists check", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "323190fa75e7aed77f65eca7ae2c5a07a6c466d7bf78865ede5c57f9ae8ffca8")
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "retailerName",
			"created_by":       "API",
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", "name", "retailerName").Return(false, errors.New("connection Timeout"))
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error while updating", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "323190fa75e7aed77f65eca7ae2c5a07a6c466d7bf78865ede5c57f9ae8ffca8")
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "retailerName",
			"created_by":       "API",
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", "name", "retailerName").Return(false, nil)
		fireStoreClient.On("Update", mock.Anything, common.RetailersCollection, "r12345", mock.Anything).Return(time.Now(), errors.New("connection timeout"))
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Update successful", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/retailers/r12345", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "323190fa75e7aed77f65eca7ae2c5a07a6c466d7bf78865ede5c57f9ae8ffca8")
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "retailerName",
			"created_by":       "API",
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		//updatedTime := time.Now().UTC().Round(time.Second)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", "name", "retailerName").Return(false, nil)
		fireStoreClient.On("Update", mock.Anything, common.RetailersCollection, "r12345", mock.Anything).Return(time.Now(), nil)
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		patchRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		//Etag := response.Header.Get(common.HeaderEtag)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Equal(t, fmt.Sprintf("{\"id\":\"RetailerID\",\"name\":\"retailerName\",\"created_by\":\"API\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"%s\"}", updateTimeStr), string(bytes))
	})
}
