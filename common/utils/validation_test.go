package utils

import (
	"context"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/retailers/models"
	sitemodel "github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/go-andiamo/urit"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestValidateHeaders(t *testing.T) {
	type args struct {
		r               *http.Request
		requiredHeaders []string
	}
	type testCase struct {
		name string
		args args
		want []string
	}

	tests := []testCase{
		{
			"All required headers are present",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderXCorrelationID,
					common.HeaderAcceptVersion),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion},
			},
			nil,
		},
		{
			"Request with valid page_token header",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageToken),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageToken},
			},
			nil,
		},
		{
			"Request with invalid page_token header",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageToken),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageToken},
			},
			[]string{"Invalid header value, unable to decrypt header : page_token"},
		},
		{
			"Request with valid page_size header",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageSize),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageSize},
			},
			nil,
		},
		{
			"Request with invalid page_size header",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageSize),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageSize},
			},
			[]string{"Unsupported value for header : page_size"},
		},
		{
			"Request with overlimit page_size header",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageSize),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderPageSize},
			},
			[]string{"page_size must be between 2 to 100"},
		},
		{
			"Only one required headers is present",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderAcceptVersion),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion},
			},
			[]string{"Request does not have the required headers : [X-Correlation-ID]"},
		},
		{
			"Only one required headers is present with wrong value",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderAcceptVersion),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion},
			},
			[]string{
				"Unsupported value for header : Accept-Version",
				"Request does not have the required headers : [X-Correlation-ID]",
			},
		},
		{
			"No required headers is present",
			args{
				getRequest(http.MethodGet, "/", ""),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion},
			},
			[]string{"Request does not have the required headers : [X-Correlation-ID Accept-Version]"},
		},
		{
			"Request with invalid retailer_id header",
			args{
				getRequest(http.MethodGet, "/", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion),
				[]string{common.HeaderXCorrelationID, common.HeaderAcceptVersion, common.HeaderRetailerID},
			},
			[]string{
				"Incorrect value for header retailer_id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Only one required headers is present with wrong value" {
				tt.args.r.Header.Set(common.HeaderAcceptVersion, "v2")
				if got := validateHeaders(tt.args.r, tt.args.requiredHeaders); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ValidateHeaders() = %v, want %v", got, tt.want)
				}
			} else if tt.name == "Request with invalid page_size header" {
				tt.args.r.Header.Set(common.HeaderPageSize, "v2")
				if got := validateHeaders(tt.args.r, tt.args.requiredHeaders); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ValidateHeaders() = %v, want %v", got, tt.want)
				}
			} else if tt.name == "Request with overlimit page_size header" {
				tt.args.r.Header.Set(common.HeaderPageSize, "200")
				if got := validateHeaders(tt.args.r, tt.args.requiredHeaders); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ValidateHeaders() = %v, want %v", got, tt.want)
				}
			} else if tt.name == "Request with invalid page_token header" {
				tt.args.r.Header.Set(common.HeaderPageToken, "invalid token")
				if got := validateHeaders(tt.args.r, tt.args.requiredHeaders); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ValidateHeaders() = %v, want %v", got, tt.want)
				}
			} else if tt.name == "Request with invalid retailer_id header" {
				tt.args.r.Header.Set(common.HeaderRetailerID, "s12345")
				if got := validateHeaders(tt.args.r, tt.args.requiredHeaders); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ValidateHeaders() = %v, want %v", got, tt.want)
				}
			} else {
				if got := validateHeaders(tt.args.r, tt.args.requiredHeaders); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ValidateHeaders() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestQueryParams(t *testing.T) {
	type args struct {
		r                   *http.Request
		requiredQueryParams []string
	}
	type testCase struct {
		name string
		args args
		want []string
	}

	tests := []testCase{
		{
			"All required query params are present",
			args{
				getRequest(http.MethodGet, "/api?q1=1&q2=2&q3=3&q4=4", ""),
				[]string{"q1", "q2", "q3", "q4"},
			},
			nil,
		},
		{
			"Only one required query params is present",
			args{
				getRequest(http.MethodGet, "/api?q1=1", ""),
				[]string{"q1", "q2", "q3", "q4"},
			},
			[]string{"Request does not have the required query params : [q2 q3 q4]"},
		},
		{
			"No required query params is present",
			args{
				getRequest(http.MethodPost, "/api", "{}"),
				[]string{"q1", "q2", "q3", "q4"},
			},
			[]string{"Request does not have the required query params : [q1 q2 q3 q4]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateQueryParams(tt.args.r, tt.args.requiredQueryParams); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRequest(t *testing.T) {
	type args struct {
		r                 *http.Request
		requestValidation RequestValidation
	}
	type testCase struct {
		name string
		args args
		want *response.Response
	}

	tests := []testCase{
		{
			"All required headers and query params are present",
			args{
				getRequest(http.MethodGet, "/api?q1=1234", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion),
				RequestValidation{
					RequiredHeaders:     []string{common.HeaderXCorrelationID, common.HeaderAcceptVersion},
					RequiredQueryParams: []string{"q1"},
					RequestMethod:       http.MethodGet,
				},
			},
			nil,
		},
		{
			"All required headers and query params are not present",
			args{
				getRequest(http.MethodDelete, "/api?q1=1234", "", common.HeaderXCorrelationID),
				RequestValidation{
					RequiredHeaders:     []string{common.HeaderXCorrelationID, common.HeaderAcceptVersion},
					RequiredQueryParams: []string{"q2"},
					RequestMethod:       http.MethodDelete,
				},
			},
			&response.Response{
				Code:    400,
				Message: "Request validation failed",
				Errors: []string{"Request does not have the required headers : [Accept-Version]",
					"Request does not have the required query params : [q2]"},
			},
		},
		{
			"Invalid Body",
			args{
				getRequest(http.MethodPost, "/retailers", "{\"id\":\"r12345\"}", common.HeaderXCorrelationID),
				RequestValidation{
					RequestMethod: http.MethodPost,
					RequestBodyValidation: &RequestBodyValidation{
						Entity:             &models.Retailer{},
						CompleteValidation: true,
					},
				},
			},
			&response.Response{
				Code:    400,
				Message: "Request body validation failed",
				Errors: []string{"Key: 'Retailer.ID' Error:Field validation for 'ID' failed on the 'disallowed' tag",
					"Key: 'Retailer.Name' Error:Field validation for 'Name' failed on the 'required' tag"},
			},
		},
		{
			"Invalid method in request",
			args{
				getRequest(http.MethodPost, "/api?q1=1234", "", common.HeaderXCorrelationID, common.HeaderAcceptVersion),
				RequestValidation{
					RequiredHeaders:     []string{common.HeaderXCorrelationID, common.HeaderAcceptVersion},
					RequiredQueryParams: []string{"q1"},
					RequestMethod:       http.MethodGet,
				},
			},
			&response.Response{
				Code:    400,
				Message: "Request validation failed",
				Errors:  []string{"Invalid request method, send request with correct method"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, got := ValidateRequest(tt.args.r, tt.args.requestValidation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateBodyComplete(t *testing.T) {
	type args struct {
		body                  io.ReadCloser
		requestBodyValidation *RequestBodyValidation
	}
	tests := []struct {
		name string
		args args
		want *response.Response
	}{
		{
			"Incorrect JSON Entity",
			args{
				getRequest(http.MethodPost, "/retailers", "{name:123}").Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: true,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Please input correct JSON in request body", []string{"invalid character 'n' looking for beginning of object key string"}),
		},
		{
			"Required fields not passed",
			args{
				getRequest(http.MethodPost, "/retailers", `{"id":"123"}`).Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: true,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Retailer.ID' Error:Field validation for 'ID' failed on the 'disallowed' tag",
					"Key: 'Retailer.Name' Error:Field validation for 'Name' failed on the 'required' tag"}),
		},
		{
			"All fields passed",
			args{
				getRequest(http.MethodPost, "/retailers", `{"name":"123456"}`).Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: true,
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, validateBody(context.Background(), tt.args.body, tt.args.requestBodyValidation),
				"validateBody( %v, %v)", tt.args.body, tt.args.requestBodyValidation.Entity)
		})
	}
}

func TestValidateBodyPartial(t *testing.T) {
	type args struct {
		body                  io.ReadCloser
		requestBodyValidation *RequestBodyValidation
	}
	tests := []struct {
		name string
		args args
		want *response.Response
	}{
		{
			"Incorrect JSON Entity",
			args{
				getRequest(http.MethodPost, "/retailers", "{name:123546}").Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Please input correct JSON in request body", []string{"invalid character 'n' looking for beginning of object key string"}),
		},
		{
			"Required fields not passed",
			args{
				getRequest(http.MethodPost, "/retailers", `{"id":"123456"}`).Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Retailer.ID' Error:Field validation for 'ID' failed on the 'disallowed' tag"}),
		},
		{
			"All fields passed",
			args{
				getRequest(http.MethodPost, "/retailers", `{"name":"123456"}`).Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: false,
				},
			},
			nil,
		},
		{
			"Incorrect JSON Entity for lat, long",
			args{
				getRequest(http.MethodPost, "/site", "{\"name\":\"sitename\",\"location\":{}}").Body,
				&RequestBodyValidation{
					Entity:             &sitemodel.Site{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Site.Location.long' Error:Field validation for 'long' failed on the 'required' tag",
					"Key: 'Site.Location.lat' Error:Field validation for 'lat' failed on the 'required' tag"}),
		},
		{
			"Incorrect JSON Entity for lat value",
			args{
				getRequest(http.MethodPost, "/site", "{\"name\":\"sitename\","+
					"\"location\":{\"lat\":110,\"long\":100}}").Body,
				&RequestBodyValidation{
					Entity:             &sitemodel.Site{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Site.Location.lat' Error:Field validation for 'lat' failed on the '-90 < lat < 90' tag"}),
		},
		{
			"Incorrect JSON Entity for long value",
			args{
				getRequest(http.MethodPost, "/site", "{\"name\":\"sitename\","+
					"\"location\":{\"lat\":10,\"long\":200}}").Body,
				&RequestBodyValidation{
					Entity:             &sitemodel.Site{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Site.Location.long' Error:Field validation for 'long' failed on the '-180 < long < 180' tag"}),
		},
		{
			"Correct name format for Retailer Name with '.'",
			args{
				getRequest(http.MethodPost, "/retailers", "{\"name\":\"123.asdad\"}").Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: false,
				},
			},
			nil,
		},
		{
			"Incorrect name format for Retailer Name",
			args{
				getRequest(http.MethodPost, "/retailers", "{\"name\":\"123.asdad.\"}").Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Retailer.Name' Error:Field validation for 'Name' failed on the 'name' tag"}),
		},
		{
			"Retailer Name exceeding max allowed characters",
			args{
				getRequest(http.MethodPost, "/retailers", "{\"name\":\"HrMEvzXkMg kcwNkdbHBk "+
					"NLyd-hbjXNd diNHHHAYzk- uRaNwNEbuV teDtpYCeva zNpQeBmffY vkhDJJzBDd QGYmKCGPfP "+
					"UpPwQycGZV WrED.xuppib ticdfrTNJX XKYcKhXXzQ nugSMNKXfG_ CNTzazDHWU wExeSAfRRh "+
					"LWrASkuYCL rVPRTNUwma iVUKmNTgqF zHRWExxFuj\"}").Body,
				&RequestBodyValidation{
					Entity:             &models.Retailer{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Retailer.Name' Error:Field validation for 'Name' failed on the 'name' tag"}),
		},
		{
			"Incorrect name format for Site Name",
			args{
				getRequest(http.MethodPost, "/sites", "{\"name\":\"site@name\","+
					"\"location\":{\"lat\":10,\"long\":179}}").Body,
				&RequestBodyValidation{
					Entity:             &sitemodel.Site{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Site.Name' Error:Field validation for 'Name' failed on the 'name' tag"}),
		},
		{
			"Site Name exceeding max allowed characters",
			args{
				getRequest(http.MethodPost, "/sites", "{\"name\":\"bfTxGbQXiM 4Qdgtrvkf6 g5qhfPW883"+
					" vqfXPGkhuL Snh4LXDfN-7 iEbwpL3Hv8_ nNMdNSnW77 Agz-F4CyRS9 2dH2D6Kg_an NkJbkHRSxG 7AMS8TzEak"+
					" JTMbgAdPq4 R6yuyYgLkB MbY65NfBN2 iw6kxZPJnL\","+"\"location\":{\"lat\":10,\"long\":179}}").Body,
				&RequestBodyValidation{
					Entity:             &sitemodel.Site{},
					CompleteValidation: false,
				},
			},
			response.NewResponse(http.StatusBadRequest, "Request body validation failed",
				[]string{"Key: 'Site.Name' Error:Field validation for 'Name' failed on the 'name' tag"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, validateBody(context.Background(), tt.args.body, tt.args.requestBodyValidation),
				"validateBody( %v, %v)", tt.args.body, tt.args.requestBodyValidation.Entity)
		})
	}
}

func TestValidateHeaderRetailerID(t *testing.T) {
	t.Run("Header Retailer ID starting with r", func(t *testing.T) {
		r := getRequest(http.MethodPost, "/retailers", "{\"name\":\"siteID1\","+
			"\"retailer_site_id\" : \"ABS111\","+
			"\"location\" : {"+
			"\"lat\" : 54.25,"+
			"\"long\" : 13.134"+
			"}"+
			"}", common.HeaderXCorrelationID)
		r.Header.Set(common.HeaderAcceptVersion, common.APIVersionV1)
		r.Header.Set(common.HeaderRetailerID, "s"+GetRandomID(4))
		validateHeaderRetailer := validateHeaderRetailerID(r, r.Header.Get(common.HeaderRetailerID))
		assert.Equal(t, "[Incorrect value for header retailer_id]", fmt.Sprint(validateHeaderRetailer))
	})
}

func getRequest(method string, url string, body string, headers ...string) *http.Request {
	request := httptest.NewRequest(method, url, strings.NewReader(body))
	for _, header := range headers {
		if header == common.HeaderAcceptVersion {
			request.Header.Set(header, common.APIVersionV1)
		} else if header == common.HeaderPageToken {
			token, _ := GetNextPageToken("r12345", common.RetailersEncryptionKey)
			request.Header.Set(header, token)
		} else if header == common.HeaderPageSize {
			request.Header.Set(header, "5")
		} else {
			request.Header.Set(header, GetRandomID(common.RandomIDLength))
		}
	}

	return request
}

func TestValidatePath(t *testing.T) {
	t.Run("Validate invalid path match", func(t *testing.T) {
		path := urit.MustCreateTemplate("/retailers/{id}")
		pathParam, err := validatePath(httptest.NewRequest(http.MethodGet, "/retailers", nil), path)
		assert.Nil(t, pathParam)
		assert.NotNil(t, err)
	})

	t.Run("Validate valid path match", func(t *testing.T) {
		path := urit.MustCreateTemplate("/retailers/{id}")
		pathParam, err := validatePath(httptest.NewRequest(http.MethodGet, "/retailers/r12345", nil), path)
		assert.NotNil(t, pathParam)
		assert.Empty(t, err)
		assert.Equal(t, "r12345", pathParam["id"])
	})

	t.Run("Validate valid path match", func(t *testing.T) {
		path := urit.MustCreateTemplate("/retailers/{id}")
		pathParam, err := validatePath(httptest.NewRequest(http.MethodGet, "/retailers/", nil), path)
		assert.Nil(t, pathParam)
		assert.NotNil(t, err)
	})
}

func TestValidateMethod(t *testing.T) {
	t.Run("Request method different", func(t *testing.T) {
		method := http.MethodPost
		err := validateMethod(httptest.NewRequest(http.MethodGet, "/retailers/", nil), method)
		assert.NotNil(t, err)
	})

	t.Run("Request method match", func(t *testing.T) {
		method := http.MethodGet
		err := validateMethod(httptest.NewRequest(http.MethodGet, "/retailers/", nil), method)
		assert.Nil(t, err)
	})
}
