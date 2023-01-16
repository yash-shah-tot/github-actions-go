package sites

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

func Test_postSite(t *testing.T) {
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
				getRequest(http.MethodPost, "/sites", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID),
			},
		},
		{
			"Request with no headers",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPost, "/sites", ""),
			},
		},
		{
			"Request with missing header retailer id",
			args{
				httptest.NewRecorder(),
				getRequest(http.MethodPost, "/sites", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postSite(tt.args.w, tt.args.request)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_postSiteHandler(t *testing.T) {
	t.Run("Invalid method request", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		method := http.MethodGet
		w := httptest.NewRecorder()
		r := getRequest(method, "/sites", "{invalid:json}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "rasdas")
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Invalid JSON body in POST", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{invalid:json}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "rasdas")
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Please input correct JSON in request body\",\"errors\":[\"invalid character 'i' looking for beginning of object key string\"]}", string(bytes))
	})

	t.Run("Missing Site Name in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"id\":\"siteID\","+
			"\"retailer_site_id\" : \"ABS134\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "rasdas")
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\""+
			",\"errors\":[\"Key: 'Site.ID' Error:Field validation for 'ID' failed on the 'disallowed' tag\","+
			"\"Key: 'Site.Name' Error:Field validation for 'Name' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Missing Retailer Site Name in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "rasdas")
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\""+
			",\"errors\":[\"Key: 'Site.RetailerSiteID' Error:Field validation for 'RetailerSiteID' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Missing Location in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID\","+
			"\"retailer_site_id\" : \"ABS134\""+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "rasdas")
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Site.Location' Error:Field validation for 'Location' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Missing Longitude in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID\","+
			"\"retailer_site_id\" : \"ABS134\","+
			"\"location\" : {"+
			"\"lat\" : 54.25"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "rasdas")
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Site.Location.long' Error:Field validation for 'long' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Missing Latitude in JSON POST Entity", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)

		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID\","+
			"\"retailer_site_id\" : \"ABS134\","+
			"\"location\" : {"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "rasdas")
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request body validation failed\","+
			"\"errors\":[\"Key: 'Site.Location.lat' Error:Field validation for 'lat' failed on the 'required' tag\"]}", string(bytes))
	})

	t.Run("Retailer ID does not exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":404,\"message\":\"Retailer ID %s not found\"}", mockedRetailerID), string(bytes))
	})

	t.Run("Site Name Already Exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(true, nil).Once()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Site with name : siteID1 already exists\"}", string(bytes))
	})

	t.Run("Retailer's Site ID Already Exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(true, nil).Once()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Retailer's site id ABS111 already exists\"}", string(bytes))
	})

	t.Run("Error while checking existence of Site name", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, errors.New("ABCD")).Once()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error while checking existence of Retailer Site name", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		w := httptest.NewRecorder()
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, errors.New("mock error")).Once()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Error while checking existence of Site ID ", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
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
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)

		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SitesCollection, mock.Anything, mock.Anything).Return(false, errors.New("mock error")).Once()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}")
	})

	t.Run("Error while saving the site details", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
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
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)

		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SitesCollection, mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Save", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(time.Now(), errors.New(mock.Anything)).Once()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}")
	})

	t.Run("Error while retrieving timezone", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
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
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()

		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Contains(t, string(bytes), "{\"code\":400,\"message\":\"Error occurred while retrieving location with latitude 54.250000 and longitude 13.134000. Timezone API returned with status: INVALID_REQUEST.")
	})

	t.Run("Site saved successfully", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
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
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "rcfsk"
		r.Header.Set(common.HeaderRetailerID, retailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()

		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SitesCollection, mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Save", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(time.Now(), nil)

		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		pubSubClient.On("Publish", mock.Anything,
			mock.Anything, mock.Anything).Return()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusCreated, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		updateTime := time.Now().UTC().Round(time.Second)
		updateTimeStr := updateTime.Format(time.RFC3339)
		assert.Contains(t, string(bytes), fmt.Sprintf("\"name\":\"siteID1\",\"retailer_site_id\":\"ABS111\",\"retailer_id\":\"rcfsk\",\"status\":\"draft\",\"timezone\":\"Europe/Berlin\",\"location\":{\"lat\":54.25,\"long\":13.134},\"created_by\":\"api@takeoff.com\",\"updated_by\":\"api@takeoff.com\",\"created_time\":\"%s\",\"updated_time\":\"%s\"}", updateTimeStr, updateTimeStr))
	})

	t.Run("Error while creating unique site id across retailers", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
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
		r := getRequest(http.MethodPost, "/sites", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		retailerID := "r" + utils.GetRandomID(4)
		r.Header.Set(common.HeaderRetailerID, retailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, retailerID, true).Return(retailer, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()
		fireStoreClient.On("Exists", mock.Anything, utils.GetSitePath(retailerID), mock.Anything, mock.Anything).Return(false, nil).Once()

		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SitesCollection, mock.Anything, mock.Anything).Return(true, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SitesCollection, mock.Anything, mock.Anything).Return(true, nil).Once()
		fireStoreClient.On("ExistsInCollectionGroup", mock.Anything, common.SitesCollection, mock.Anything, mock.Anything).Return(true, nil).Once()
		postSiteHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, string(bytes), "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}")
	})
}
