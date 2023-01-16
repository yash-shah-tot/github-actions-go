package spokes

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
	"time"
)

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
}

func Test_patchSiteAttachSpoke(t *testing.T) {
	mockedSiteID := "s" + utils.GetRandomID(4)
	mockedSpokeID := "p" + utils.GetRandomID(4)
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
				httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchSpokeAttach(tt.args.w, tt.args.r)
			assert.Equal(t, tt.args.w.Result().StatusCode, http.StatusBadRequest)
		})
	}
}

func Test_patchSpokeAttachHandler(t *testing.T) {
	t.Run("Invalid method request", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		method := http.MethodPost
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":400,\"message\":\"Request validation failed\",\"errors\":[\"Invalid request method, send request with correct method\"]}", string(bytes))
	})

	t.Run("Retailer ID does not exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, status.Error(codes.NotFound, "Retailer ID not found"))
		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":404,\"message\":\"Retailer ID %s not found\"}", mockedRetailerID), string(bytes))
	})

	t.Run("Site ID does not exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mock.Anything, true).Return(nil, status.Error(codes.NotFound, "Site ID not found"))
		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":404,\"message\":\"Site ID %s not found\"}", mockedSiteID), string(bytes))
	})

	t.Run("Spoke ID does not exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mock.Anything, true).Return(site, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, true).Return(nil, status.Error(codes.NotFound, "Spoke ID not found"))

		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusNotFound, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":404,\"message\":\"Spoke ID %s not found\"}", mockedSpokeID), string(bytes))
	})

	t.Run("Association already exists", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		spoke := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mock.Anything, true).Return(site, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, true).Return(spoke, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(true, nil)
		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":400,\"message\":\"Spoke %s is already attached to site %s\"}", mockedSpokeID, mockedSiteID), string(bytes))
	})

	t.Run("Error while checking the association and mapping", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		spoke := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mock.Anything, true).Return(site, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, true).Return(spoke, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(true, errors.New("connection timeout"))
		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":500,\"message\":\"Internal server error occurred."+
			" Please check logs for more details.\"}"), string(bytes))
	})

	t.Run("Error while attaching spoke to site", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		spoke := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mock.Anything, true).Return(site, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, true).Return(spoke, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil)
		fireStoreClient.On("Save", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(time.Time{}, status.Error(codes.NotFound, fmt.Sprintf("Spoke ID %s is not attached to site %s", mockedSpokeID, mockedSiteID)))
		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, "{\"code\":500,\"message\":\"Internal server error occurred. Please check logs for more details.\"}", string(bytes))
	})

	t.Run("Site Spoke attached successful", func(t *testing.T) {
		pubSubClient := mocks.NewQueue(t)
		fireStoreClient := mocks.NewDB(t)
		mockedRetailerID := "r" + utils.GetRandomID(4)
		mockedSiteID := "s" + utils.GetRandomID(4)
		mockedSpokeID := "p" + utils.GetRandomID(4)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/sites/%s/spokes/%s:attach", mockedSiteID, mockedSpokeID), nil)
		r.Header.Set(common.HeaderXCorrelationID, "123")
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, mockedRetailerID)
		retailer := map[string]interface{}{}
		site := map[string]interface{}{}
		spoke := map[string]interface{}{}
		fireStoreClient.On("GetByID", mock.Anything, common.RetailersCollection, mockedRetailerID, true).Return(retailer, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSitePath(mockedRetailerID), mock.Anything, true).Return(site, nil)
		fireStoreClient.On("GetByID", mock.Anything, utils.GetSpokePath(mockedRetailerID), mock.Anything, true).Return(spoke, nil)
		fireStoreClient.On("Exists", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(false, nil)
		fireStoreClient.On("Save", mock.Anything, utils.GetSiteSpokePath(mockedRetailerID), mock.Anything, mock.Anything).Return(time.Now(), nil)
		pubSubClient.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		patchSpokeAttachHandler(w, r, fireStoreClient, pubSubClient)
		response := w.Result()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, fmt.Sprintf("{\"code\":200,\"message\":\"Spoke %s attached successfully to site %s\"}", mockedSpokeID, mockedSiteID), string(bytes))
	})
}
