package response

import (
	"errors"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getRequest(method string, url string, body string, headers ...string) *http.Request {
	request := httptest.NewRequest(method, url, strings.NewReader(body))
	for _, header := range headers {
		if header == common.HeaderAcceptVersion {
			request.Header.Set(header, common.APIVersionV1)
		} else {
			request.Header.Set(header, "r1s4ee5")
		}
	}

	return request
}

func TestGetCommonResponseHeaders(t *testing.T) {
	type args struct {
		request *http.Request
	}
	tests := []struct {
		name          string
		args          args
		headerMapKeys []string
	}{
		{
			"Request with correlation ID header",
			args{
				getRequest(http.MethodGet, "/api?q1=1&q2=2&q3=3&q4=4", "", common.HeaderXCorrelationID),
			},
			[]string{
				common.HeaderContentType, common.HeaderXCorrelationID,
			},
		},
		{
			"Request without correlation ID header",
			args{
				getRequest(http.MethodGet, "/api?q1=1&q2=2&q3=3&q4=4", "", common.HeaderAcceptVersion),
			},
			[]string{
				common.HeaderContentType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCommonResponseHeaders(tt.args.request)
			for _, key := range tt.headerMapKeys {
				assert.True(t, got[key] != "")
			}
		})
	}
}

func TestHeaderMap_WithHeader(t *testing.T) {
	var headers = GetCommonResponseHeaders(getRequest(http.MethodGet, "/", "")).WithHeader("key", "value")
	assert.Equal(t, headers["key"], "value")
}

func TestRespondWithInternalServerError(t *testing.T) {
	t.Run("RespondWithInternalServerError", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		RespondWithInternalServerError(response, request)
		result := response.Result()
		assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
		assert.Equal(t, common.ContentTypeApplicationJSON, result.Header.Get(common.HeaderContentType))
	})
}

func TestRespondWithNotFoundErrorMessage(t *testing.T) {
	t.Run("RespondWithNotFoundErrorMessage", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		message := "Not found"
		RespondWithNotFoundErrorMessage(response, request, message, errors.New(message))
		result := response.Result()
		data, _ := io.ReadAll(result.Body)
		assert.Equal(t, http.StatusNotFound, result.StatusCode)
		assert.Equal(t, common.ContentTypeApplicationJSON, result.Header.Get(common.HeaderContentType))
		assert.Equal(t, "{\"code\":404,\"message\":\"Not found\"}", string(data))
	})
}

func TestNewResponse(t *testing.T) {
	t.Run("NewResponse", func(t *testing.T) {
		errorList := []string{"Test for Errors", "Valid Errors"}
		response := NewResponse(http.StatusNotFound, "Item Not Found", errorList)
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Equal(t, "Item Not Found", response.Message)
		assert.Equal(t, "Test for Errors", response.Errors[0])
	})
}

func TestRespondWithResponseObject(t *testing.T) {
	t.Run("Invalid Response Object", func(t *testing.T) {
		response := httptest.NewRecorder()
		Respond(response, http.StatusOK, map[string]interface{}{
			"foo": make(chan int),
		}, nil)
		bytes, _ := io.ReadAll(response.Body)
		assert.Equal(t, response.Code, http.StatusInternalServerError)
		assert.Equal(t, bytes, []byte("Unable to serialize response body"))
	})
}
