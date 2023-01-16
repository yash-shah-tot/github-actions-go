package sites

import (
	"errors"
	"fmt"
	commonModels "github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/h2non/gock"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var retailer = map[string]interface{}{}

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
	retailer = map[string]interface{}{
		"id":               "RetailerID",
		"name":             "RetailerName",
		"created_by":       common.User,
		"updated_by":       "",
		"deactivated_by":   "",
		"created_time":     "2022-10-28T07:33:05Z",
		"updated_time":     nil,
		"deactivated_time": "2022-10-28T07:33:05Z",
	}
}

func Test_patchSite(t *testing.T) {
	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Request with no headers",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPatch, "/sites/s12345", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchSite(tt.args.w, tt.args.r)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}
func Test_patchSiteHandler(t *testing.T) {
	t.Run("SitesID not passed in path param", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"EOF\"]}", string(bytes))
	})

	t.Run("Invalid method request", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodPost
		w := httptest.NewRecorder()
		r := getRequest(method, "/sites/s12345", "{invalid:json}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Invalid JSON body in PATCH", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{invalid:json}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"invalid character 'i' looking for beginning of object key string\"]}", string(bytes))
	})

	t.Run("Missing required fields in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"Empty JSON received, please input valid JSON in body\"]}", string(bytes))
	})

	t.Run("lat not found in body", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"North site 2 updadfd\",\"location\":{\"long\":13.134}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Site.Location.lat' Error:Field validation for 'lat' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("lat,long not found in body", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"North site 2 updadfd\",\"location\":{}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Site.Location.long' Error:Field validation for 'long' failed on the 'required' tag\","+
			"\"Key: 'Site.Location.lat' Error:Field validation for 'lat' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("lat value is out of bounds in body", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"North site 2 updadfd\",\"location\":{\"lat\":113.134,\"long\":13.134}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\",\"errors\":[\"Key: 'Site.Location.lat' Error:Field validation for 'lat' failed on the '-90 \\u003c lat \\u003c 90' tag\"]}", string(bytes))
	})

	t.Run("extra filed in input body", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"North site 2 updadfd\",\"location\":{\"lat\":154.25,\"long\":13.134},\"extras\":\"extra\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"json: unknown field \\\"extras\\\"\"]}", string(bytes))
	})

	t.Run("SitesID does not exist", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderIfMatch, "1f818230eecabdefa93c35acaa7807a7df2dc1935e24d106ba5bab6368601f3a")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(nil, status.Error(codes.NotFound, "Site ID s12345 not found"))
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Site ID s12345 not found\"}", string(bytes))
	})

	t.Run("Failed while getting data from DB", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderIfMatch, "1f818230eecabdefa93c35acaa7807a7df2dc1935e24d106ba5bab6368601f3a")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(nil, errors.New("connection timeout"))
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Missing Longitude in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345",
			"{\"location\":{\"lat\" : 53.001}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderIfMatch, "1f818230eecabdefa93c35acaa7807a7df2dc1935e24d106ba5bab6368601f3a")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Site.Location.long' Error:Field validation for 'long' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Retailer ID does not exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteID1\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderIfMatch, "1f818230eecabdefa93c35acaa7807a7df2dc1935e24d106ba5bab6368601f3a")
		mockedRetailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, status.Error(codes.NotFound, fmt.Sprintf("Retailer ID %s not found", mockedRetailerID)))
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":404,\"message\":\"Retailer ID %s not found\"}", mockedRetailerID), string(bytes))
	})

	t.Run("Retailer ID is not found internal error", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderIfMatch, "962c6f4e95a51388b55a53fa1b212ad3a7d748f3a3cf6e30a5d1cba3caf2dac8")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, status.Error(codes.Internal, "Site ID s12345 not found"))
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("ETag Mismatch", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set(common.HeaderIfMatch, "1f818230eecabdefa93c35acaa7807a7df2dc1935e24d106ba5bab6368601443")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "siteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusPreconditionFailed, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":412,\"message\":\"If-Match header value incorrect, please get the latest and try again\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		site := map[string]interface{}{
			"foo": make(chan int),
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error occurred during json unmarshalling invalid site returned from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		pubSubClient := mocks.NewQueue(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "6c910040c3bd9bf90b07ea06a369b0db18e112cd0d3c068c361139424f6a5363")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		site := map[string]interface{}{
			"id": 1234,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Site name already exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "159f536fdb35efdb1c1f25d780c1082fbdccdb4bb298d2faab07efa597770099")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "siteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), "name", "siteName").Return(true, nil)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":422,\"message\":\"Site with name : siteName already exists\"}", string(bytes))
	})

	t.Run("DB Down during exists check", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "159f536fdb35efdb1c1f25d780c1082fbdccdb4bb298d2faab07efa597770099")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "siteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), "name", "siteName").Return(false, errors.New("connection Timeout"))
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error while updating", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"newSiteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "159f536fdb35efdb1c1f25d780c1082fbdccdb4bb298d2faab07efa597770099")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "siteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), "name", "newSiteName").Return(false, nil)
		fireStoreClient.On("Update", mock.Anything, utils.GetSitePath("r12345"), "s12345", mock.Anything).Return(time.Now(), errors.New("connection timeout"))
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Update name successful", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\"}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "6e02e0a0289fbdc09e30919785ba0ffa055fe507e1a2271e585bb73fae92f218")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		defer gock.Off()
		gock.New("https:/maps.googleapis.com").
			Get("/maps/api/timezone/json").
			Reply(200).
			JSON(commonModels.GoogleTimeZone{
				DstOffset:    0,
				RawOffset:    3600,
				Status:       "OK",
				TimezoneID:   "Europe/Berlin",
				TimezoneName: "Central European Standard Time",
			})
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "oldSiteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), "name", "siteName").Return(false, nil)
		fireStoreClient.On("Update", mock.Anything, utils.GetSitePath("r12345"), "s12345", mock.Anything).Return(time.Now(), nil)
		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Equal(t, fmt.Sprintf("{\"id\":\"s12345\",\"name\":\"siteName\",\"retailer_site_id\":\"ABC123\",\"retailer_id\":\"r12345\",\"status\":\"\",\"timezone\":\"\",\"location\":null,\"created_by\":\"API\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"%s\"}", updateTimeStr), string(bytes))
	})

	t.Run("Update location successful", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"location\":{\"lat\":40.730610,\"long\":-73.935242}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "6e02e0a0289fbdc09e30919785ba0ffa055fe507e1a2271e585bb73fae92f218")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		defer gock.Off()
		gock.New("https:/maps.googleapis.com").
			Get("/maps/api/timezone/json").
			Reply(200).
			JSON(commonModels.GoogleTimeZone{
				DstOffset:    0,
				RawOffset:    3600,
				Status:       "OK",
				TimezoneID:   "Europe/Berlin",
				TimezoneName: "Central European Standard Time",
			})
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "oldSiteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}

		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Update", mock.Anything, utils.GetSitePath("r12345"), "s12345", mock.Anything).Return(time.Now(), nil)
		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Equal(t, fmt.Sprintf("{\"id\":\"s12345\",\"name\":\"oldSiteName\",\"retailer_site_id\":\"ABC123\",\"retailer_id\":\"r12345\",\"status\":\"\",\"timezone\":\"Europe/Berlin\",\"location\":{\"lat\":40.73061,\"long\":-73.935242},\"created_by\":\"API\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"%s\"}", updateTimeStr), string(bytes))
	})

	t.Run("Update same location failed", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"location\":{\"lat\":40.730610,\"long\":-73.935242}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "6e02e0a0289fbdc09e30919785ba0ffa055fe507e1a2271e585bb73fae92f218")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		defer gock.Off()
		gock.New("https:/maps.googleapis.com").
			Get("/maps/api/timezone/json").
			Reply(200).
			JSON(commonModels.GoogleTimeZone{
				DstOffset:    0,
				RawOffset:    3600,
				Status:       "OK",
				TimezoneID:   "Europe/Berlin",
				TimezoneName: "Central European Standard Time",
			})
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "oldSiteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
		}

		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Update", mock.Anything, utils.GetSitePath("r12345"), "s12345", mock.Anything).Return(time.Now(), nil)
		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Equal(t, fmt.Sprintf("{\"id\":\"s12345\",\"name\":\"oldSiteName\",\"retailer_site_id\":\"ABC123\",\"retailer_id\":\"r12345\",\"status\":\"\",\"timezone\":\"Europe/Berlin\",\"location\":{\"lat\":40.73061,\"long\":-73.935242},\"created_by\":\"API\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"%s\"}", updateTimeStr), string(bytes))
	})

	t.Run("Update same location failed", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"location\":{\"lat\":40.730610,\"long\":-73.935242}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "affd6a3f9581077470f579b26a6f14289f0d33dc35e674273fc5e48b50f53d47")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		defer gock.Off()
		gock.New("https:/maps.googleapis.com").
			Get("/maps/api/timezone/json").
			Reply(200).
			JSON(commonModels.GoogleTimeZone{
				DstOffset:    0,
				RawOffset:    3600,
				Status:       "OK",
				TimezoneID:   "Europe/Berlin",
				TimezoneName: "Central European Standard Time",
			})
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "siteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
			"location": map[string]interface{}{
				"lat":  40.730610,
				"long": -73.935242,
			},
		}

		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":422,\"message\":\"No changes detected\"}", string(bytes))
	})

	t.Run("Update all successful", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"New SiteName\",\"location\":{\"lat\":40.730610,\"long\":-73.935242}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "affd6a3f9581077470f579b26a6f14289f0d33dc35e674273fc5e48b50f53d47")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		defer gock.Off()
		gock.New("https:/maps.googleapis.com").
			Get("/maps/api/timezone/json").
			Reply(200).
			JSON(commonModels.GoogleTimeZone{
				DstOffset:    0,
				RawOffset:    3600,
				Status:       "OK",
				TimezoneID:   "Europe/Berlin",
				TimezoneName: "Central European Standard Time",
			})
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "siteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
			"location": map[string]interface{}{
				"lat":  40.730610,
				"long": -73.935242,
			},
		}

		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), "name", "New SiteName").Return(false, nil)
		fireStoreClient.On("Update", mock.Anything, utils.GetSitePath("r12345"), "s12345", mock.Anything).Return(time.Now(), nil)
		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Equal(t, fmt.Sprintf("{\"id\":\"s12345\",\"name\":\"New SiteName\",\"retailer_site_id\":\"ABC123\",\"retailer_id\":\"r12345\",\"status\":\"\",\"timezone\":\"\",\"location\":{\"lat\":40.73061,\"long\":-73.935242},\"created_by\":\"API\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"%s\"}", updateTimeStr), string(bytes))
	})

	t.Run("Error while retrieving timezone", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		defer gock.Off()
		gock.New("https:/maps.googleapis.com").
			Get("/maps/api/timezone/json").
			Reply(200).
			JSON(commonModels.GoogleTimeZone{
				DstOffset:    0,
				RawOffset:    3600,
				Status:       "INVALID_REQUEST",
				TimezoneID:   "Europe/Berlin",
				TimezoneName: "Central European Standard Time",
			})
		w := httptest.NewRecorder()
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "siteName",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"created_by":       "API",
			"updated_by":       "API",
			"deactivated_by":   "",
			"created_time":     "2022-10-28T07:33:05Z",
			"updated_time":     "2022-10-28T07:33:05Z",
			"deactivated_time": nil,
			"location": map[string]interface{}{
				"lat":  40.730610,
				"long": -73.935242,
			},
		}
		r := getRequest(http.MethodPatch, "/sites/s12345", "{\"name\":\"siteName\",\"location\":{\"lat\":54.250000,\"long\":13.134000}}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		r.Header.Set("If-Match", "affd6a3f9581077470f579b26a6f14289f0d33dc35e674273fc5e48b50f53d47")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		// TODO if running test individually then please uncomment this line
		//secretsManagerClient.On("GetSecretValue", mock.Anything, mock.Anything).Return("ABCXYZ", nil)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath("r12345"), "s12345", true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), "name", "siteName").Return(false, nil)

		patchSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		defer response.Body.Close()
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":400,\"message\":\"Error occurred while retrieving location with"+
			" latitude 54.250000 and longitude 13.134000. Timezone API returned with status: INVALID_REQUEST. Please provide valid location details\"}")
	})
}
