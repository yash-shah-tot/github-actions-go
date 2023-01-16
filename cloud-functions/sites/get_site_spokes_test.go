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

func Test_getSiteSpokes(t *testing.T) {
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
				getRequest(http.MethodGet, "/sites/s12345/spokes", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getSiteSpokes(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_getSiteSpokeHandler(t *testing.T) {
	t.Run("Required headers not passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			fmt.Sprintf("{\"code\":400,"+
				"\"message\":\"Request validation failed\","+
				"\"errors\":["+
				"\"Request does not have the required headers : [%s %s %s]\"]}",
				common.HeaderAcceptVersion, common.HeaderXCorrelationID, common.HeaderRetailerID),
			string(bytes))
	})

	t.Run("Invalid method request", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodPost
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Required headers passed with invalid page token header", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageToken, "abcxyz")
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t,
			fmt.Sprintf("{\"code\":400,"+
				"\"message\":\"Request validation failed\","+
				"\"errors\":["+
				"\"Invalid header value, unable to decrypt header : page_token\"]}"),
			string(bytes))
	})

	t.Run("Required headers passed with page size header passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSiteSpokeList(10), "p12345", nil).Once()
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSpokesList(10), "p12345", nil).Once()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.NotEmpty(t, response.Header.Get("next_page_token"))
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Required headers passed with page size header passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSiteSpokeList(10), "p12345", nil).Once()
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSpokesList(5), "p12345", nil).Once()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Empty(t, response.Header.Get("next_page_token"))
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Return empty list", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSiteSpokeList(0), "p12345", nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Passed invalid next page token", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		r.Header.Set(common.HeaderPageToken, "invalid")

		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid header value, unable to decrypt header : page_token\"]}", string(bytes))
	})

	t.Run("Passed invalid page_size header", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "invalid")
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Unsupported value for header : page_size\"]}", string(bytes))
	})

	t.Run("Passed valid next page token", func(t *testing.T) {
		token, _ := utils.GetNextPageToken("p12345", common.SpokesEncryptionKey)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSiteSpokeList(10), "p12345", nil).Once()
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSpokesList(10), "p12345", nil).Once()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageToken, token)
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("RetailerID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("SiteID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil).Once()
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, "s12345", true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found")).Once()
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Site ID s12345 not found\"}", string(bytes))
	})

	t.Run("page size beyond limit", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "200")
		token, _ := utils.GetNextPageToken("1234-12334", common.SpokesEncryptionKey)
		r.Header.Set(common.HeaderPageToken, token)
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"page_size must be between 2 to 100\"]}", string(bytes))
	})

	t.Run("DB returned wrong entities", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getBadSite(), "r12345", nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageSize, "10")
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Passed valid next page token with deactivated=true", func(t *testing.T) {
		token, _ := utils.GetNextPageToken("r12345", common.SitesEncryptionKey)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSiteSpokeList(1), "p12345", nil).Once()
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSpokesList(1), "p12345", nil).Once()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes?deactivated=true", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageToken, token)
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("DB error while fetching site spoke mapping", func(t *testing.T) {
		token, _ := utils.GetNextPageToken("r12345", common.SitesEncryptionKey)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, "", errors.New("connection timeout")).Once()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageToken, token)
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("DB error while fetching spokes for the site", func(t *testing.T) {
		token, _ := utils.GetNextPageToken("r12345", common.SitesEncryptionKey)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("GetByID", mock.Anything, mock.Anything, mock.Anything, true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(getSiteSpokeList(10), "p12345", nil).Once()
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, "", errors.New("connection timeout")).Once()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sites/s12345/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageToken, token)
		getSiteSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})
}

func getSiteSpokeList(length int) []map[string]interface{} {
	var list []map[string]interface{}
	for i := 0; i < length; i++ {
		data := map[string]interface{}{
			"site_id":  utils.GetRandomID(5),
			"spoke_id": utils.GetRandomID(5),
		}
		list = append(list, data)
	}

	return list
}

func getSpokesList(length int) []map[string]interface{} {
	var list []map[string]interface{}
	for i := 0; i < length; i++ {
		data := map[string]interface{}{
			"id":          "p12345",
			"name":        "spoke 8",
			"retailer_id": "r485sh",
			"timezone":    "Europe/Bucharest",
			"location": map[string]interface{}{
				"lat":  45.394,
				"long": 23.844,
			},
			"created_by":   common.User,
			"updated_by":   common.User,
			"created_time": "2022-11-24T05:41:47Z",
			"updated_time": "2022-11-24T05:41:47Z",
		}
		list = append(list, data)
	}

	return list
}
