package sites

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

func Test_getSiteAudit(t *testing.T) {
	type args struct {
		w       *httptest.ResponseRecorder
		request *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Request with no retailer_id header",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getSiteAudit(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_getSiteAuditHandler(t *testing.T) {
	t.Run("Invalid method request", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodPost
		w := httptest.NewRecorder()
		r := getRequest(method, "/sites/s12345/auditLogs", "{invalid:json}", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID)
		getSiteAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("RetailerID not passed in header and site_id not in path param", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites", nil)
		getSiteAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Request does not have the required headers : [Accept-Version X-Correlation-ID retailer_id]\",\"Invalid request url path, no matching path params found in path : /sites\"]}", string(bytes))
	})

	t.Run("RetailerID and Site ID combination does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), common.ID, "s12345").Return(false, nil)
		getSiteAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Site with id s12345 does not exist for Retailer with id r12345\"}", string(bytes))
	})

	t.Run("Failed while checking existence of retailer id from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), common.ID, "s12345").Return(false, errors.New("connection timeout"))
		getSiteAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Failed while getting audit logs from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), common.ID, "s12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, "", errors.New("connection timeout"))
		getSiteAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Successful get audit logs from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), common.ID, "s12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getSiteAuditList(1), "", nil)
		getSiteAuditHandler(w, r, fireStoreClient)
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
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "2")
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), common.ID, "s12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getSiteAuditList(2), "12345-12345", nil)
		getSiteAuditHandler(w, r, fireStoreClient)
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
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "2")
		token, _ := utils.GetNextPageToken("1234-12334", common.SitesEncryptionKey)
		r.Header.Set(common.HeaderPageToken, token)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), common.ID, "s12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getSiteAuditList(2), "123-1234", nil)
		getSiteAuditHandler(w, r, fireStoreClient)
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

	t.Run("page size beyond limit", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "500")
		token, _ := utils.GetNextPageToken("1234-12334", common.SitesEncryptionKey)
		r.Header.Set(common.HeaderPageToken, token)
		getSiteAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"page_size must be between 2 to 100\"]}", string(bytes))
	})

	t.Run("Got invalid audit logs from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodGet, "/sites/s12345/auditLogs", "", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath("r12345"), common.ID, "s12345").Return(true, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getBadSiteAudit(), "", nil)
		getSiteAuditHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})
}

func getSiteAuditList(length int) []map[string]interface{} {
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

func getBadSiteAudit() []map[string]interface{} {
	var list []map[string]interface{}

	data := map[string]interface{}{
		"id":   make(chan int),
		"name": utils.GetRandomID(5),
	}
	list = append(list, data)

	return list
}
