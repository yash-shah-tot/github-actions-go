package retailers

import (
	"errors"
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

func Test_deactivateRetailer(t *testing.T) {
	type args struct {
		w       *httptest.ResponseRecorder
		request *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Request with no headers",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPost, "/retailers/r12345:deactivate", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRetailerDeactivate(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_postRetailerDeactivate(t *testing.T) {
	t.Run("RetailerID not passed in path param", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request url path, no matching path params found in path : /retailers/\"]}", string(bytes))
	})

	t.Run("Deactivate not passed in path param", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request url path, no matching path params found in path : /retailers/r12345\"]}", string(bytes))
	})

	t.Run("Invalid Request Method", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		method := http.MethodPatch
		w := httptest.NewRecorder()
		r := getRequest(method, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("RetailerID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 does not exist\"}", string(bytes))
	})

	t.Run("Failed while getting data from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, errors.New("connection timeout"))
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Deactivating already deactivated retailer", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "b75cf21766ceaf71a61befba5740f83d69e5d1af143e915257e442f780387fdc")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(nil, status.Error(codes.NotFound, "document not found"))
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 does not exist\"}", string(bytes))
	})

	t.Run("Check sub-documents returns error", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("CheckSubDocuments", mock.Anything, mock.Anything, "r12345").Return(false, errors.New("error while fetching the retailer from DB"))
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Deleting retailer with active sites", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("CheckSubDocuments", mock.Anything, mock.Anything, "r12345").Return(false, nil)
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusPreconditionFailed, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":412,\"message\":\"Deactivate request cannot be processed, there are active sites under the said retailer.\"}", string(bytes))
	})

	t.Run("ETag Mismatch", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("CheckSubDocuments", mock.Anything, mock.Anything, "r12345").Return(true, nil)
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusPreconditionFailed, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":412,\"message\":\"If-Match header value incorrect, please get the latest and try again\"}", string(bytes))
	})

	t.Run("Error while Deleting from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "6e6bca71b47fc4ff92a54f7ca00e693e11962713008fb4c84026773db245f325")
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("CheckSubDocuments", mock.Anything, mock.Anything, "r12345").Return(true, nil)
		fireStoreClient.On("Update", mock.Anything, common.RetailersCollection, "r12345", mock.Anything).Return(time.Now(), errors.New("connection timeout"))
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Successful Deleting from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "6e6bca71b47fc4ff92a54f7ca00e693e11962713008fb4c84026773db245f325")
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("CheckSubDocuments", mock.Anything, mock.Anything, "r12345").Return(true, nil)
		fireStoreClient.On("Update", mock.Anything, common.RetailersCollection, "r12345", mock.Anything).Return(time.Now(), nil)
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":200,\"message\":\"Retailer r12345 deactivated successfully\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		retailer := map[string]interface{}{
			"foo": make(chan int),
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("CheckSubDocuments", mock.Anything, mock.Anything, "r12345").Return(true, nil)
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling invalid retailer returned", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345:deactivate", "",
			common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch)
		r.Header.Set("If-Match", "185a5203b0ed48bd8b816a8355aa98bc2bbf370ca200b42ddc752c241c673c2a")
		retailer := map[string]interface{}{
			"id": 123,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("CheckSubDocuments", mock.Anything, mock.Anything, "r12345").Return(true, nil)
		postRetailerDeactivateHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})
}
