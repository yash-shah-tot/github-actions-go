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
)

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
}

func Test_getRetailer(t *testing.T) {
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
				getRequest(http.MethodGet, "/retailers/r12345", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getRetailer(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_getRetailerHandler(t *testing.T) {
	t.Run("Invalid Request Method", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers/r12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("RetailerID not passed in path param", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/retailers", nil)
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Request does not have the required headers : [Accept-Version X-Correlation-ID]\",\"Invalid request url path, no matching path params found in path : /retailers\"]}", string(bytes))
	})

	t.Run("RetailerID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("Failed while getting data from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, errors.New("connection timeout"))
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Successful Get of Retailer", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       common.User,
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "ea298f4a6944e09b6efbc16fa411aea791cef2db874d4192f2ddbfdbabaf1622", response.Header.Get(common.HeaderEtag))
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"id\":\"RetailerID\",\"name\":\"RetailerName\",\"created_by\":\"api@takeoff.com\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"2022-10-28T07:33:05Z\"}", string(bytes))
	})

	t.Run("Retailer ID is deleted and deleted not passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(nil, status.Error(codes.NotFound, "document not found"))
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("Retailer ID deleted and deleted=true passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, fmt.Sprintf("/retailers/r12345?%s=%s", common.QueryParamDeactivated, common.True), "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       common.User,
			"deactivated_by":   common.User,
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": "2022-10-28T07:33:05Z",
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", false).Return(retailer, nil)
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "0399ddda6721928f06205e1cacc66391fd1ae08cd173add37e433e4fa4ffa948", response.Header.Get(common.HeaderEtag))
		assert.Equal(t, "{\"id\":\"RetailerID\",\"name\":\"RetailerName\",\"created_by\":\"api@takeoff.com\",\"updated_by\":\"api@takeoff.com\",\"deactivated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"2022-10-28T07:33:05Z\",\"deactivated_time\":\"2022-10-28T07:33:05Z\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"foo": make(chan int),
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling invalid retailer returned", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id": 1233,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		getRetailerHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})
}
