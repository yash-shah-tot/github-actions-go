package spokes

import (
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
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

var retailer = map[string]interface{}{}

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
}

func Test_getSpokes(t *testing.T) {
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
				httptest.NewRequest(http.MethodGet, "/spokes", nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getSpokes(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_getSpokesHandler(t *testing.T) {
	t.Run("Required headers not passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		getSpokesHandler(w, r, fireStoreClient)
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
		r := httptest.NewRequest(method, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "2")
		token, _ := utils.GetNextPageToken("1234-12334", common.SitesEncryptionKey)
		r.Header.Set(common.HeaderPageToken, token)
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Required headers passed with invalid page token header", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageToken, "abcde")
		getSpokesHandler(w, r, fireStoreClient)
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

	t.Run("Passed invalid page_size header", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "invalid")
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Unsupported value for header : page_size\"]}", string(bytes))
	})

	t.Run("page size beyond limit", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderPageSize, "200")
		token, _ := utils.GetNextPageToken("1234-12334", common.SitesEncryptionKey)
		r.Header.Set(common.HeaderPageToken, token)
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"page_size must be between 2 to 100\"]}", string(bytes))
	})

	t.Run("RetailerID does not exist", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":404,\"message\":\"Retailer ID r12345 not found\"}", string(bytes))
	})

	t.Run("Retailer ID is not found internal error", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, status.Error(codes.Internal, "not found"))
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Required headers passed with page size header passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where{{
			Field:    common.DeactivatedTime,
			Operator: common.OperatorEquals,
			Value:    nil,
		}}).
			Return(getSpokeList(10), "r12345", nil) //returning list equal to page size
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.NotEmpty(t, response.Header.Get("next_page_token"))
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Required headers passed with page size header passed", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where{{
			Field:    common.DeactivatedTime,
			Operator: common.OperatorEquals,
			Value:    nil,
		}}).
			Return(getSpokeList(5), "r12345", nil) //returning list less than page size
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Empty(t, response.Header.Get("next_page_token"))
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Return empty list", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where{{
			Field:    common.DeactivatedTime,
			Operator: common.OperatorEquals,
			Value:    nil,
		}}).
			Return(getSpokeList(0), "r12345", nil)
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Passed invalid next page token", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderPageSize, "10")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where{{
			Field:    common.DeactivatedTime,
			Operator: common.OperatorEquals,
			Value:    nil,
		}}).
			Return(getSpokeList(5), "r12345", nil)
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Passed valid next page token", func(t *testing.T) {
		token, _ := utils.GetNextPageToken("r12345", common.SitesEncryptionKey)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageToken, token)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where{{
			Field:    common.DeactivatedTime,
			Operator: common.OperatorEquals,
			Value:    nil,
		}}).
			Return(getSpokeList(5), "r12345", nil)
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Passed valid next page token with deactivated=true", func(t *testing.T) {
		token, _ := utils.GetNextPageToken("r12345", common.SitesEncryptionKey)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes?deactivated=true", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageToken, token)
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where(nil)).
			Return(getSpokeList(5), "r12345", nil)
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("DB returned wrong entities", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes?deactivated=true", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		r.Header.Set(common.HeaderPageSize, "10")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where(nil)).
			Return(getBadSpoke(), "r12345", nil)
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.NotEmpty(t, bytes)
	})

	t.Run("Error fetching from DB", func(t *testing.T) {
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/spokes", nil)
		r.Header.Set(common.HeaderXCorrelationID, "1234")
		r.Header.Set(common.HeaderRetailerID, "r12345")
		r.Header.Set(common.HeaderAcceptVersion, "v1")
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, "r12345", true).Return(retailer, nil)
		fireStoreClient.On("GetAll", mock.Anything, mock.Anything, mock.Anything, []cloud.Where{{
			Field:    common.DeactivatedTime,
			Operator: common.OperatorEquals,
			Value:    nil,
		}}).
			Return(nil, "", errors.New("connection timeout"))
		getSpokesHandler(w, r, fireStoreClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	})
}

func getSpokeList(length int) []map[string]interface{} {
	var list []map[string]interface{}
	for i := 0; i < length; i++ {
		data := map[string]interface{}{
			"id":   utils.GetRandomID(5),
			"name": utils.GetRandomID(5),
		}
		list = append(list, data)
	}

	return list
}

func getBadSpoke() []map[string]interface{} {
	var list []map[string]interface{}

	data := map[string]interface{}{
		"id":   make(chan int),
		"name": utils.GetRandomID(5),
	}
	list = append(list, data)

	return list
}
