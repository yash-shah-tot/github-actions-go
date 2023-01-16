package retailers

import (
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/TakeoffTech/site-info-svc/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
}

func getRequest(method string, url string, body string, headers ...string) *http.Request {
	request := httptest.NewRequest(method, url, strings.NewReader(body))
	for _, header := range headers {
		if header == common.HeaderAcceptVersion {
			request.Header.Set(header, common.APIVersionV1)
		} else {
			request.Header.Set(header, utils.GetRandomID(common.RandomIDLength))
		}
	}

	return request
}

func Test_postRetailer(t *testing.T) {
	type args struct {
		w       *httptest.ResponseRecorder
		request *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Request with all required headers",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPost, "/retailers", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion),
			},
		},
		{
			"Request with no headers",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPost, "/retailers", ""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRetailer(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_postRetailerHandler(t *testing.T) {
	t.Run("Invalid Request Method", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodGet
		w := httptest.NewRecorder()
		r := getRequest(method, "/retailers", "{}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Invalid JSON body in POST", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{invalid:json}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"invalid character 'i' looking for beginning of object key string\"]}", string(bytes))
	})

	t.Run("Missing required fields in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{\"id\":\"retailerID\"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\""+
			",\"errors\":[\"Key: 'Retailer.ID' Error:Field validation for 'ID' failed on the 'disallowed' tag\","+
			"\"Key: 'Retailer.Name' Error:Field validation for 'Name' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Retailer name already exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", "name", "retailerName").Return(true, nil)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Retailer with name : retailerName already exists\"}", string(bytes))
	})

	t.Run("DB Down during name exists check", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", "name", "retailerName").Return(false, errors.New("connection Timeout"))
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Retailer ID Already Exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(true, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Save", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).
			Return(time.Now(), errors.New("conflict ID already exists"))
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error while checking Retailer ID Exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(false, errors.New("connection Timeout"))
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Retailer Saved Successfully", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(false, nil)
		fireStoreClient.On("Save", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(time.Now(), nil)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusCreated, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Contains(t, string(bytes), fmt.Sprintf("\"name\":\"retailerName\",\"created_by\":\"api@takeoff.com\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"%s\",\"updated_time\":\"%s\"}", updateTimeStr, updateTimeStr))
	})

	t.Run("Unable to create a unique id for retailer", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, "site-info-retailers", mock.Anything, mock.Anything).Return(true, nil)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/retailers", "{\"name\":\"retailerName\"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		postRetailerHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})
}
