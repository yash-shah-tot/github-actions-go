package sites

import (
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/utils"
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

func Test_getSite(t *testing.T) {
	type args struct {
		w       *httptest.ResponseRecorder
		request *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Request with no headers ",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodGet, "/sites/s12345", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getSite(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_getSiteHandler(t *testing.T) {
	t.Run("SiteID not passed in path param", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites", nil)
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Request does not have the required headers : [Accept-Version X-Correlation-ID retailer_id]\",\"Invalid request url path, no matching path params found in path : /sites\"]}", string(bytes))
	})

	t.Run("Invalid method request", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodPost
		w := httptest.NewRecorder()
		r := getRequest(method, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("RetailerID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("Failed while getting retailer data from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, errors.New("connection timeout"))
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Retailer ID is deleted", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(nil, status.Error(codes.NotFound, "document not found"))
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("Site ID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   common.User,
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, status.Error(codes.NotFound, "Site ID not found"))
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Site ID s12345 not found\"}", string(bytes))
	})

	t.Run("Failed while getting site data from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   common.User,
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, errors.New("connection timeout"))
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   common.User,
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		site := map[string]interface{}{
			"foo": make(chan int),
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Site is deleted and deleted not passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   common.User,
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(nil, status.Error(codes.NotFound, "document not found"))
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Site ID s12345 not found\"}", string(bytes))
	})

	t.Run("Site ID deleted and deleted=true passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, fmt.Sprintf("/sites/s12345?%s=%s", common.QueryParamDeactivated, common.True), "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":             "RetailerID",
			"name":           "RetailerName",
			"created_by":     common.User,
			"updated_by":     common.User,
			"deactivated_by": common.User,
			"created_time":   "2022-10-28T07:33:05Z",
			"updated_time":   "2022-10-28T07:33:05Z",
		}
		site := map[string]interface{}{
			"id":               "sd4d9r",
			"name":             "site 8",
			"retailer_site_id": "ABS141",
			"retailer_id":      "r485sh",
			"status":           "DRAFT",
			"timezone":         "Europe/Bucharest",
			"location": map[string]interface{}{
				"lat":  45.394,
				"long": 23.844,
			},
			"created_by":       common.User,
			"updated_by":       common.User,
			"created_time":     "2022-11-24T05:41:47Z",
			"deactivated_time": "2022-11-28T07:33:05Z",
			"updated_time":     "2022-11-24T05:41:47Z",
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", false).Return(site, nil)
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"id\":\"sd4d9r\",\"name\":\"site 8\",\"retailer_site_id\":\"ABS141\",\"retailer_id\":\"r485sh\",\"status\":\"DRAFT\",\"timezone\":\"Europe/Bucharest\",\"location\":{\"lat\":45.394,\"long\":23.844},\"created_by\":\"api@takeoff.com\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-11-24T05:41:47Z\",\"updated_time\":\"2022-11-24T05:41:47Z\",\"deactivated_time\":\"2022-11-28T07:33:05Z\"}", string(bytes))
	})

	t.Run("Site fetched successfully", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailer := map[string]interface{}{
			"id":               "RetailerID",
			"name":             "RetailerName",
			"created_by":       common.User,
			"updated_by":       "",
			"deactivated_by":   common.User,
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     nil,
			"deactivated_time": nil,
		}
		site := map[string]interface{}{
			"id":               "sd4d9r",
			"name":             "site 8",
			"retailer_site_id": "ABS141",
			"retailer_id":      "r485sh",
			"status":           "DRAFT",
			"timezone":         "Europe/Bucharest",
			"location": map[string]interface{}{
				"lat":  45.394,
				"long": 23.844,
			},
			"created_by":   common.User,
			"updated_by":   common.User,
			"created_time": "2022-11-24T05:41:47Z",
			"updated_time": "2022-11-24T05:41:47Z",
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		getSiteHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"id\":\"sd4d9r\",\"name\":\"site 8\",\"retailer_site_id\":\"ABS141\",\"retailer_id\":\"r485sh\",\"status\":\"DRAFT\",\"timezone\":\"Europe/Bucharest\",\"location\":{\"lat\":45.394,\"long\":23.844},\"created_by\":\"api@takeoff.com\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-11-24T05:41:47Z\",\"updated_time\":\"2022-11-24T05:41:47Z\"}", string(bytes))
	})
}
