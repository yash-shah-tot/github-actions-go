package response

import (
	"encoding/json"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"net/http"
)

type HeaderMap map[string]string

// WithHeader will return an updated HeaderMap with the provided headerKey and headerValue
func (headerMap HeaderMap) WithHeader(headerKey string, headerValue string) HeaderMap {
	responseHeaders := headerMap
	if headerValue != "" {
		responseHeaders[headerKey] = headerValue
	}

	return responseHeaders
}

type Response struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

// NewResponse Creates and returns a new Response Object
func NewResponse(code int, message string, errors []string) *Response {
	if len(errors) > 0 {
		return newResponseWithErrors(code, message, errors)
	}

	return &Response{Code: code, Message: message, Errors: nil}
}

func newResponseWithErrors(code int, message string, errors []string) *Response {
	return &Response{Code: code, Message: message, Errors: errors}
}

// Respond function will take http.ResponseWriter and other response related inputs
// It will assign statusCode, responseHeaders and write the response object value in response
func Respond(responseWriter http.ResponseWriter, statusCode int, response any, responseHeaders map[string]string) {
	encoded, err := json.Marshal(response)
	if err != nil {
		responseWriter.WriteHeader(http.StatusInternalServerError)
		_, _ = responseWriter.Write([]byte("Unable to serialize response body"))

		return
	}

	for headerKey, headerValue := range responseHeaders {
		responseWriter.Header().Add(headerKey, headerValue)
	}

	responseWriter.WriteHeader(statusCode)
	_, _ = responseWriter.Write(encoded)
}
func RespondWithResponseObject(responseWriter http.ResponseWriter, response *Response,
	responseHeaders map[string]string) {
	Respond(responseWriter, response.Code, response, responseHeaders)
}

// RespondWithInternalServerError will create response with internal server error status and given error message
func RespondWithInternalServerError(responseWriter http.ResponseWriter, request *http.Request) {
	RespondWithResponseObject(responseWriter, NewResponse(http.StatusInternalServerError,
		"Internal server error occurred. "+
			"Please check logs for more details.", nil),
		GetCommonResponseHeaders(request))
}

// RespondWithNotFoundErrorMessage will create response with not found status and given error message
// also will log the error message.
func RespondWithNotFoundErrorMessage(responseWriter http.ResponseWriter, request *http.Request,
	message string, err error) {
	logger := logging.GetLoggerFromContext(request.Context())
	logger.Debugf("%s : %v", message, err)
	RespondWithResponseObject(responseWriter, NewResponse(http.StatusNotFound, message, nil),
		GetCommonResponseHeaders(request))
}

// GetCommonResponseHeaders will return a HeaderMap of the common headers
func GetCommonResponseHeaders(request *http.Request) HeaderMap {
	responseHeaders := make(map[string]string)
	responseHeaders[common.HeaderContentType] = common.ContentTypeApplicationJSON
	if request.Header.Get(common.HeaderXCorrelationID) != "" {
		responseHeaders[common.HeaderXCorrelationID] = request.Header.Get(common.HeaderXCorrelationID)
	}

	return responseHeaders
}
