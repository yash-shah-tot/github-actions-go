package sites

import (
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
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

var siteStatus = map[string]interface{}{
	"id": common.SiteStatusTransitionsDocument,
	"status-transitions": map[string]interface{}{
		"draft":               []string{"provisioning"},
		"active":              []string{"inactive"},
		"inactive":            []string{"active", "deprovisioning"},
		"provisioning":        []string{"inactive", "provisioning-failed"},
		"deprovisioning":      []string{"deprecated"},
		"provisioning-failed": []string{"deprovisioning", "provisioning"},
		"deprecated":          []string{},
	},
}

var invalidSiteStatus = map[string]interface{}{
	"id": make(chan int),
}

var site map[string]interface{}

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
	site = map[string]interface{}{
		"id":               "s12345",
		"name":             "site name",
		"retailer_site_id": "r site id",
		"retailer_id":      "r12345",
		"status":           "draft",
		"timezone":         "UTC",
		"location": map[string]interface{}{
			"lat":  10.12,
			"long": 10.12,
		},
		"created_by":       "user",
		"updated_by":       "user",
		"deactivated_by":   "",
		"created_time":     "2022-10-28T07:33:05Z",
		"updated_time":     "2022-10-28T07:33:05Z",
		"deactivated_time": nil,
	}
}

func Test_patchSiteStatus(t *testing.T) {
	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Request with all required headers but no status",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s", "s12345"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID),
			},
		},
		{
			"Request with invalid path",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPatch, fmt.Sprintf("/retailers/%s:%s", "s12345", "inactive"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchSiteStatus(tt.args.w, tt.args.r)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_patchSiteStatusHandler(t *testing.T) {
	t.Run("Invalid method request", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodPost
		w := httptest.NewRecorder()
		r := getRequest(method, fmt.Sprintf("/sites/%s:%s", "s12345", "status"), "{invalid:json}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderIfMatch, common.HeaderRetailerID)
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Valid request path with missing if-match header", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "status"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID)
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			fmt.Sprintf("{\"code\":400,"+
				"\"message\":\"Request validation failed\","+
				"\"errors\":["+
				"\"Request does not have the required headers : [%s]\"]}",
				common.HeaderIfMatch),
			string(bytes))
	})

	t.Run("Valid request path error while fetching status transition from DB", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "status"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(nil, errors.New("connection timeout"))
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}",
			string(bytes))
	})

	t.Run("Invalid status document received from DB", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "status"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(invalidSiteStatus, nil)
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}",
			string(bytes))
	})

	t.Run("Invalid target status received in the request", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "start"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil)
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":400,\"message\":\"Invalid status 'start' received in the request\"}",
			string(bytes))
	})

	t.Run("Site ID not found in the DB", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "provisioning"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(nil, status.Error(codes.NotFound, "not found")).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":404,\"message\":\"Site ID s12345 not found\"}",
			string(bytes))
	})

	t.Run("Error while getting site from the DB", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "provisioning"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(nil, errors.New("connection  timeout")).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}",
			string(bytes))
	})

	t.Run("Error while computing etag site from the DB", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "provisioning"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(invalidSiteStatus, nil).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}",
			string(bytes))
	})

	t.Run("Etag Mismatch", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "provisioning"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(site, nil).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusPreconditionFailed, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":412,\"message\":\"If-Match header value incorrect, please get the latest and try again\"}", string(bytes))
	})

	t.Run("Invalid site object returned from DB", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "provisioning"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		r.Header.Set(common.HeaderIfMatch, "6c910040c3bd9bf90b07ea06a369b0db18e112cd0d3c068c361139424f6a5363")
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		wrongSite := map[string]interface{}{
			"id": 1234,
		}
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(wrongSite, nil).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}",
			string(bytes))
	})

	t.Run("Site status corrupted in the DB", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "provisioning"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		r.Header.Set(common.HeaderIfMatch, "1f7fc5cb29d00be448b1941eb13958892627a544d134d42fc2af4e3eb4ffff4b")
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		corruptSite := map[string]interface{}{
			"status": "invalid",
		}
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(corruptSite, nil).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}",
			string(bytes))
	})

	t.Run("Invalid target status transition", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "draft"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		r.Header.Set(common.HeaderIfMatch, "ffcc9870a751a0241f5f2bdac8e6646c40b92bb226e8efc4af2e29cc242fc176")
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		siteInActive := map[string]interface{}{
			"status": "active",
		}
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(siteInActive, nil).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":400,\"message\":\"Invalid status transition received in the request. The site status cannot be changed from active to draft status\"}",
			string(bytes))
	})

	t.Run("Error while doing the update", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "inactive"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		r.Header.Set(common.HeaderIfMatch, "ffcc9870a751a0241f5f2bdac8e6646c40b92bb226e8efc4af2e29cc242fc176")
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		siteInActive := map[string]interface{}{
			"status": "active",
		}
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(siteInActive, nil).Once()
		fireStoreClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Now(), errors.New("update failed")).Once()
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			"{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}",
			string(bytes))
	})

	t.Run("Successful update for active", func(t *testing.T) {
		cleanUp()
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "active"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		r.Header.Set(common.HeaderIfMatch, "059370dd3b618a7a6ed4cb056d35df805b7709f1614dd3a7cc8b6ef9e61f4131")
		fireStoreClient.On("GetByID", mock.Anything, common.StatusTransitionsCollection,
			common.SiteStatusTransitionsDocument, false).Return(siteStatus, nil).Once()
		siteInDeprovisioning := site
		siteInDeprovisioning["status"] = "inactive"
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(siteInDeprovisioning, nil).Once()
		fireStoreClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Now(), nil).Once()
		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf(
			"{\"id\":\"s12345\",\"name\":\"site name\",\"retailer_site_id\":\"r site id\",\"retailer_id\":\"r12345\",\"status\":\"active\",\"timezone\":\"UTC\",\"location\":{\"lat\":10.12,\"long\":10.12},\"created_by\":\"user\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"%s\"}", updateTimeStr), string(bytes))
	})

	t.Run("Successful update for deprecated with cached site statuses", func(t *testing.T) {
		// This test case won't run individually, you will have to run it as part of the suite only
		// This is because we are testing the caching scenario so the earlier test case needs
		// to be run before this test case
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPatch, fmt.Sprintf("/sites/%s:%s", "s12345", "deprecated"), "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID, common.HeaderIfMatch)
		r.Header.Set(common.HeaderIfMatch, "2c2e4ae0ab881b9165d42e92027b11a420f8ed59d81457c8e61cb3dc98d5daf2")
		siteInDeprovisioning := site
		siteInDeprovisioning["status"] = "deprovisioning"
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything,
			"s12345", true).Return(siteInDeprovisioning, nil).Once()
		fireStoreClient.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(time.Now(), nil).Once()
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		patchSiteStatusHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf(
			"{\"id\":\"s12345\",\"name\":\"site name\",\"retailer_site_id\":\"r site id\",\"retailer_id\":\"r12345\",\"status\":\"deprecated\",\"timezone\":\"UTC\",\"location\":{\"lat\":10.12,\"long\":10.12},\"created_by\":\"user\",\"updated_by\":\"api@takeoff.com\",\"deactivated_by\":\"api@takeoff.com\",\"created_time\":\"2022-10-28T07:33:05Z\",\"updated_time\":\"%s\",\"deactivated_time\":\"%s\"}", updateTimeStr, updateTimeStr), string(bytes))
	})
}

func cleanUp() {
	siteStatuses = models.SiteStatuses{}
}
