package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	commonModels "github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/fatih/structs"
	"github.com/go-andiamo/urit"
	"github.com/go-playground/validator/v10"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var validate = validator.New()

func init() {
	_ = validate.RegisterValidation("disallowed", validateDisallowed, true)
	validate.RegisterStructValidation(validateLocation, &commonModels.Location{})
	_ = validate.RegisterValidation("name", validateName)
}

type RequestBodyValidation struct {
	Entity             interface{}
	CompleteValidation bool
}

type RequestValidation struct {
	RequiredHeaders       []string
	RequiredQueryParams   []string
	RequiredPath          urit.Template
	RequestMethod         string
	RequestBodyValidation *RequestBodyValidation
}

// ValidateRequest is a common method to validate all types of requests
func ValidateRequest(request *http.Request,
	requestValidation RequestValidation) (map[string]string, *response.Response) {
	pathParams, validatePathError := validatePath(request, requestValidation.RequiredPath)
	validateMethodErrors := validateMethod(request, requestValidation.RequestMethod)
	validateHeaderErrors := validateHeaders(request, requestValidation.RequiredHeaders)
	validateQueryParamErrors := validateQueryParams(request, requestValidation.RequiredQueryParams)
	validationErrors := append(validateHeaderErrors, validateQueryParamErrors...)
	validationErrors = append(validationErrors, validateMethodErrors...)
	validationErrors = append(validationErrors, validatePathError...)
	if validationErrors != nil {
		return pathParams, &response.Response{
			Code:    http.StatusBadRequest,
			Message: "Request validation failed",
			Errors:  validationErrors,
		}
	}
	validateBodyResponse := validateBody(request.Context(), request.Body, requestValidation.RequestBodyValidation)
	if validateBodyResponse != nil {
		return pathParams, validateBodyResponse
	}

	return pathParams, nil
}

// ValidateBody is a common method to validate body of requests according to the entity object passed
// This will decode the request body and convert it into struct
// further validating it based on struct validate tag using validator
func validateBody(ctx context.Context, body io.ReadCloser,
	requestBodyValidation *RequestBodyValidation) *response.Response {
	if requestBodyValidation == nil {
		return nil
	}
	jsonDecoder := json.NewDecoder(body)
	jsonDecoder.DisallowUnknownFields()
	err := jsonDecoder.Decode(requestBodyValidation.Entity)
	if err != nil || structs.IsZero(requestBodyValidation.Entity) {
		return jsonDecodeErrors(ctx, err, requestBodyValidation)
	}
	if requestBodyValidation.CompleteValidation {
		err = validate.StructCtx(ctx, requestBodyValidation.Entity)
	} else {
		validationFields := GetValidationFields(requestBodyValidation.Entity)
		err = validate.StructPartialCtx(ctx, requestBodyValidation.Entity, validationFields...)
	}

	if err != nil {
		var errs []string
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			for _, e := range valErrs {
				errs = append(errs, e.Error())
			}
		}

		return &response.Response{
			Code:    http.StatusBadRequest,
			Message: "Request body validation failed",
			Errors:  errs,
		}
	}

	return nil
}

func validateDisallowed(fieldLevel validator.FieldLevel) bool {
	field := fieldLevel.Field()

	return field.IsZero()
}

func validateHeaders(request *http.Request, requiredHeaders []string) []string {
	var errs []string
	var missingHeaders []string
	for _, header := range requiredHeaders {
		if request.Header.Get(header) == "" {
			missingHeaders = append(missingHeaders, header)
		} else {
			switch header {
			case common.HeaderAcceptVersion:
				errs = append(errs, validateAcceptVersion(request, header)...)
			case common.HeaderPageSize:
				errs = append(errs, validatePageSize(request, header)...)
			case common.HeaderPageToken:
				errs = append(errs, ValidatePageToken(request, header)...)
			case common.HeaderRetailerID:
				errs = append(errs, validateHeaderRetailerID(request, header)...)
			}
		}
	}
	if len(missingHeaders) > 0 {
		errs = append(errs, fmt.Sprintf("Request does not have the required headers : %v", missingHeaders))
	}

	return errs
}

func validateQueryParams(request *http.Request, requiredQueryParams []string) []string {
	var errs []string
	var missingQueryParams []string
	for _, queryParam := range requiredQueryParams {
		if !request.URL.Query().Has(queryParam) {
			missingQueryParams = append(missingQueryParams, queryParam)
		}
	}
	if len(missingQueryParams) > 0 {
		errs = append(errs, fmt.Sprintf("Request does not have the required query params : %v", missingQueryParams))
	}

	return errs
}

func GetValidationFields(data interface{}) []string {
	var fields []string
	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if !v.Field(i).IsZero() {
			fields = append(fields, t.Field(i).Name)
		}
	}

	return fields
}

func validateHeaderRetailerID(request *http.Request, header string) []string {
	var errs []string
	if !strings.HasPrefix(request.Header.Get(header), common.RetailerIDPrefix) {
		errs = append(errs, "Incorrect value for header retailer_id")
	}

	return errs
}

func validateAcceptVersion(request *http.Request, header string) []string {
	var errs []string
	if !Contains(common.GetSupportedVersions(), request.Header.Get(header)) {
		errs = append(errs, fmt.Sprintf("Unsupported value for header : %v", header))
	}

	return errs
}

func validatePageSize(request *http.Request, header string) []string {
	var errs []string
	pageSize, err := strconv.Atoi(request.Header.Get(header))
	if err != nil {
		errs = append(errs, fmt.Sprintf("Unsupported value for header : %v", header))
	} else if pageSize > common.MaxPageSize || pageSize < common.MinPageSize {
		errs = append(errs, fmt.Sprintf("%s must be between %d to %d",
			common.HeaderPageSize, common.MinPageSize, common.MaxPageSize))
	}

	return errs
}

func validateLocation(structLevel validator.StructLevel) {
	location, _ := structLevel.Current().Interface().(commonModels.Location)
	if location.Longitude == nil {
		structLevel.ReportError("", "long", "", "required", "")
	}
	if location.Latitude == nil {
		structLevel.ReportError("", "lat", "", "required", "")
	}
	if location.Longitude != nil && (*location.Longitude < -180 || *location.Longitude > 180) {
		structLevel.ReportError("", "long", "", "-180 < long < 180", "")
	}
	if location.Latitude != nil && (*location.Latitude < -90 || *location.Latitude > 90) {
		structLevel.ReportError("", "lat", "", "-90 < lat < 90", "")
	}
}

func validateName(fieldLevel validator.FieldLevel) bool {
	nameValue := fieldLevel.Field().String()
	lengthNameValue := len([]rune(nameValue))

	if lengthNameValue <= common.MinNameLength || lengthNameValue > common.MaxNameLength {
		return false
	}

	regex := regexp.MustCompile(common.NameRegex)

	return regex.MatchString(nameValue)
}

func validatePath(request *http.Request, apiPath urit.Template) (map[string]string, []string) {
	pathParams := make(map[string]string)
	var errs []string
	if apiPath == nil {
		return pathParams, errs
	}
	result, match := apiPath.Matches(request.URL.Path)
	if !match {
		return nil,
			append(errs, fmt.Sprintf("Invalid request url path, no matching path params found in path : %s", request.URL.Path))
	}
	pathVars := result.GetAll()
	for _, pathVar := range pathVars {
		value, ok := pathVar.Value.(string)
		if value == "" || !ok {
			errs = append(errs,
				fmt.Sprintf("Invalid request url path, no valid matching path params found in path for %s", pathVar.Name))
		}
		pathParams[pathVar.Name] = value
	}

	return pathParams, errs
}

func validateMethod(request *http.Request, method string) []string {
	var errs []string
	if request.Method != method {
		errs = append(errs, "Invalid request method, send request with correct method")
	}

	return errs
}

func jsonDecodeErrors(ctx context.Context, err error, requestBodyValidation *RequestBodyValidation) *response.Response {
	var errs []string
	logging.GetLoggerFromContext(ctx).Debugf("Error occurred while converting json body to struct : %v", err)
	if err != nil {
		errs = append(errs, err.Error())
	}
	if err == nil && structs.IsZero(requestBodyValidation.Entity) {
		errs = append(errs, "Empty JSON received, please input valid JSON in body")
	}

	return &response.Response{
		Code:    http.StatusBadRequest,
		Message: "Please input correct JSON in request body",
		Errors:  errs,
	}
}
