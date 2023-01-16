package retailers

import (
	"encoding/json"
	"errors"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit/models"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/TakeoffTech/site-info-svc/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func Test_getRetailerAudit(t *testing.T) {
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
				getRequest(http.MethodGet, "/retailers/r12345/auditLogs", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getRetailerAudit(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_getRetailerAuditHandler(t *testing.T) {
	t.Run("Invalid Request Method", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodPost
		w := httptest.NewRecorder()
		r := getRequest(method, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("RetailerID not passed in path param", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/retailers", nil)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Request does not have the required headers : [Accept-Version X-Correlation-ID]\",\"Invalid request url path, no matching path params found in path : /retailers\"]}", string(bytes))
	})

	t.Run("RetailerID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, common.RetailersCollection, common.ID, "r12345").Return(false, nil)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer with id : r12345 does not exist\"}", string(bytes))
	})

	t.Run("Failed while checking existence of retailer id from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, common.RetailersCollection, common.ID, "r12345").Return(false, errors.New("connection timeout"))
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("page size beyond limit", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "400")
		token, _ := utils.GetNextPageToken("1234-12334", common.SitesEncryptionKey)
		r.Header.Set(common.HeaderPageToken, token)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"page_size must be between 2 to 100\"]}", string(bytes))
	})

	t.Run("Failed while getting audit logs from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, common.RetailersCollection, common.ID, "r12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, "", errors.New("connection timeout"))
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Successful get audit logs from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, common.RetailersCollection, common.ID, "r12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getRetailerAuditList(1), "", nil)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		var auditEntities []models.AuditLog
		json.Unmarshal(bytes, &auditEntities)
		assert.Equal(t, 1, len(auditEntities))
		assert.NotEmpty(t, auditEntities[0].ChangeType)
		assert.NotEmpty(t, auditEntities[0].ChangedBy)
		assert.NotEmpty(t, auditEntities[0].ChangeDetails)
		assert.NotEmpty(t, auditEntities[0].ChangedAt)
	})

	t.Run("Successful get audit logs from DB with page token in response", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "2")
		fireStoreClient.On("Exists", mock.Anything, common.RetailersCollection, common.ID, "r12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getRetailerAuditList(2), "12345-12345", nil)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		var auditEntities []models.AuditLog
		json.Unmarshal(bytes, &auditEntities)
		assert.Equal(t, 2, len(auditEntities))
		assert.NotEmpty(t, auditEntities[0].ChangeType)
		assert.NotEmpty(t, auditEntities[0].ChangedBy)
		assert.NotEmpty(t, auditEntities[0].ChangeDetails)
		assert.NotEmpty(t, auditEntities[0].ChangedAt)
		assert.NotEmpty(t, response.Header.Get(common.HeaderNextPageToken))
	})

	t.Run("Successful get audit logs from DB with page token in header", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "2")
		token, _ := utils.GetNextPageToken("1234-12334", common.RetailersEncryptionKey)
		r.Header.Set(common.HeaderPageToken, token)
		fireStoreClient.On("Exists", mock.Anything, common.RetailersCollection, common.ID, "r12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getRetailerAuditList(2), "123-1234", nil)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		var auditEntities []models.AuditLog
		json.Unmarshal(bytes, &auditEntities)
		assert.Equal(t, 2, len(auditEntities))
		assert.NotEmpty(t, auditEntities[0].ChangeType)
		assert.NotEmpty(t, auditEntities[0].ChangedBy)
		assert.NotEmpty(t, auditEntities[0].ChangeDetails)
		assert.NotEmpty(t, auditEntities[0].ChangedAt)
		assert.NotEmpty(t, response.Header.Get(common.HeaderNextPageToken))
	})

	t.Run("Got invalid audit logs from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/retailers/r12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, common.RetailersCollection, common.ID, "r12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getBadRetailerAudit(), "", nil)
		getRetailerAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})
}

func getRetailerAuditList(length int) []map[string]interface{} {
	var list []map[string]interface{}
	for i := 0; i < length; i++ {
		data := map[string]interface{}{
			"changed_by":  utils.GetRandomID(5),
			"change_type": utils.GetRandomID(5),
			"change_details": []map[string]interface{}{
				{
					"field":     utils.GetRandomID(5),
					"old_value": utils.GetRandomID(5),
					"new_value": utils.GetRandomID(5),
				},
			},
			"changed_at": time.Now().UTC().Round(time.Second),
		}
		list = append(list, data)
	}

	return list
}

func getBadRetailerAudit() []map[string]interface{} {
	var list []map[string]interface{}

	data := map[string]interface{}{
		"id":   make(chan int),
		"name": utils.GetRandomID(5),
	}
	list = append(list, data)

	return list
}
