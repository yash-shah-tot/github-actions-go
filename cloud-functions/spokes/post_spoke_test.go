package spokes

import (
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	commonModels "github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/TakeoffTech/site-info-svc/mocks"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		} else if header == common.HeaderRetailerID {
			headerRetailerID := "r" + utils.GetRandomID(5)
			request.Header.Set(header, headerRetailerID)
		} else {
			request.Header.Set(header, utils.GetRandomID(common.RandomIDLength))
		}
	}

	return request
}

func Test_postSpoke(t *testing.T) {
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
				getRequest(http.MethodPost, "/sites/s12345/spokes", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID),
			},
		},
		{
			"Request with no headers",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPost, "/sites/s12345/spokes", ""),
			},
		},
		{
			"Request with missing header retailer id",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPost, "/sites/s12345/spokes", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postSpoke(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_postSpokeHandler(t *testing.T) {
	t.Run("Invalid JSON body in POST", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites/s12345/spokes", "{invalid:json}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"invalid character 'i' looking for beginning of object key string\"]}", string(bytes))
	})

	t.Run("Invalid method request", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodGet
		w := httptest.NewRecorder()
		r := getRequest(method, "/sites/s12345/spokes", "{invalid:json}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Missing Spoke Name in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites/s12345/spokes", "{\"id\":\"spokeID\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\""+
			",\"errors\":[\"Key: 'Spoke.ID' Error:Field validation for 'ID' failed on the 'disallowed' tag\","+
			"\"Key: 'Spoke.Name' Error:Field validation for 'Name' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Missing Location in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites/s12345/spokes", "{\"name\":\"spokeName\""+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Spoke.Location' Error:Field validation for 'Location' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Missing Longitude in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites/s12345/spokes", "{\"name\":\"spokeID\","+
			"\"location\" : {"+
			"\"lat\" : 54.25"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Spoke.Location.long' Error:Field validation for 'long' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Missing Latitude in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites/s12345/spokes", "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "r12345")
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Spoke.Location.lat' Error:Field validation for 'lat' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Retailer ID does not exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites/s12345/spokes", "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":404,\"message\":\"Retailer ID %s not found\"}", mockedRetailerID), string(bytes))
	})

	t.Run("Site ID does not exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, status.Error(codes.NotFound, "Site ID not found"))
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":404,\"message\":\"Site ID %s not found\"}", mockedSiteID), string(bytes))
	})

	t.Run("Spoke Name Already Exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(true, nil).Once()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Spoke with name : spokeName already exists\"}", string(bytes))
	})

	t.Run("Error while checking existence of Spoke name", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(true, errors.New("internal server error")).Once()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error while retrieving timezone", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
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
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Contains(t, string(bytes), "{\"code\":400,\"message\":\"Error occurred while retrieving location with latitude 54.250000 and longitude 13.134000. Timezone API returned with status: INVALID_REQUEST.")
	})

	t.Run("Error while checking existence of Spoke ID", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
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
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SpokesCollection, common.ID, mock.Anything).Return(false, errors.New("mock error")).Once()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}")
	})

	t.Run("Error while checking saving Spoke in DB", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
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
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SpokesCollection, common.ID, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Save", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(time.Now(), errors.New(mock.Anything)).Once()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}")
	})

	t.Run("Error while checking saving SiteSpoke association in DB", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
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
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SpokesCollection, common.ID, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Save", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(time.Now(), nil).Once()
		fireStoreClient.On("Save", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(time.Now(), errors.New(mock.Anything)).Once()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}")
	})

	t.Run("Error while creating unique spoke id across retailers", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
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
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SpokesCollection, common.ID, mock.Anything).Return(true, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SpokesCollection, common.ID, mock.Anything).Return(true, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SpokesCollection, common.ID, mock.Anything).Return(true, nil).Once()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}")
	})

	t.Run("Site saved successfully", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedSiteID := "s" + utils.GetRandomID(5)
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
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, fmt.Sprintf("/sites/%s/spokes", mockedSiteID), "{\"name\":\"spokeName\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(5)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mockedSiteID, true).Return(site, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SpokesCollection, common.ID, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Save", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(time.Now(), nil).Once()
		fireStoreClient.On("Save", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(time.Now(), nil).Once()
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		postSpokeHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusCreated, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Contains(t, string(bytes), fmt.Sprintf("\"name\":\"spokeName\",\"retailer_id\":\"%s\",\"timezone\":\"Europe/Berlin\",\"location\":{\"lat\":54.25,\"long\":13.134},\"created_by\":\"api@takeoff.com\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"%s\",\"updated_time\":\"%s\"}", mockedRetailerID, updateTimeStr, updateTimeStr))
	})
}
