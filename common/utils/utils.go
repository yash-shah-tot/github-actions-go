package utils

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/TakeoffTech/site-info-svc/common/response"
	"github.com/fatih/structs"
	"github.com/hashicorp/packer-plugin-sdk/random"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"
)

func GetETag(data interface{}) (string, error) {
	switch reflect.TypeOf(data).Kind().String() {
	case "map":
		byteArray, err := json.Marshal(data)
		if err != nil {
			return "", err
		}

		return computeEtag(byteArray), nil
	case "struct":
		return GetETag(structs.Map(data))
	default:
		return "", errors.New("type of data did not match map or struct, returning empty etag")
	}
}

func PopulateETags(data []map[string]interface{}, object interface{}) error {
	for _, dataMap := range data {
		etag, err := GetETag(dataMap)
		if err != nil {
			return err
		}
		dataMap[common.ETag] = etag
	}

	return ConvertToObject(data, object)
}

// GetEtag accepts []byte will compute the etag and return the has in form of string
func computeEtag(data []byte) string {
	hash := sha256.Sum256(data)

	return fmt.Sprintf("%x", hash)
}

// Contains will return true if the s []string array contains the string str else false
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// ConvertToObject is a generic method which will convert the map to the passed object type
func ConvertToObject(data interface{}, object interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return unmarshal(bytes, object)
}

// GetRandomID common function to generate random alphanumeric ID of length 5
func GetRandomID(length int) string {
	return random.AlphaNumLower(length)
}

func unmarshal(bytes []byte, object interface{}) error {
	err := json.Unmarshal(bytes, object)
	if err != nil {
		return err
	}

	return nil
}

// GetSpanName will take the spanName as input and give you
// ServiceName. prefixed value which can be used for tracing
func GetSpanName(spanName string) string {
	return fmt.Sprintf("%s.%s", common.ServiceName, spanName)
}

// GetTimeZone will get time zone from the lat, log
func GetTimeZone(context context.Context, latitude float64, longitude float64) (*models.GoogleTimeZone, error) {
	apiURL, err := url.Parse(common.TimezoneAPIUrl)
	if err != nil {
		return nil, err
	}
	queryParam := apiURL.Query()
	queryParam.Set(common.LocationParam, fmt.Sprintf("%f,%f", latitude, longitude))
	queryParam.Set(common.TimestampParam, fmt.Sprintf("%d", time.Now().Unix()))
	queryParam.Set(common.APIKeyParam, os.Getenv(common.GoogleMapsAPIEnv))
	apiURL.RawQuery = queryParam.Encode()
	request, err := http.NewRequestWithContext(context, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var googleTimeZone *models.GoogleTimeZone
	if err = json.Unmarshal(body, &googleTimeZone); err != nil {
		return googleTimeZone, err
	}

	if googleTimeZone.Status != "OK" {
		err = fmt.Errorf("error : google Status : %s", googleTimeZone.Status)

		return googleTimeZone, err
	}

	return googleTimeZone, err
}

// CreateResponseForGetAllByModel common response for the get all by Model
// last parameter is object of type which is to be added in array.
func CreateResponseForGetAllByModel[T any](ctx context.Context, responseWriter http.ResponseWriter,
	request *http.Request, data []map[string]interface{}, nextPageToken string, v T) {
	logger := logging.GetLoggerFromContext(ctx)
	var modelArray []T
	err := PopulateETags(data, &modelArray)
	if err != nil {
		logger.Errorf("Error while populating etags and converting to struct object : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return
	}
	var modelArr []T
	if modelArray != nil {
		modelArr = modelArray
	} else {
		modelArr = []T{}
	}

	response.Respond(responseWriter, http.StatusOK, modelArr,
		response.GetCommonResponseHeaders(request).WithHeader(common.HeaderNextPageToken, nextPageToken))
	logger.Debugf("%d %T successfully fetched from DB", len(modelArray), v)
}

// GetSitePath will return the firestore path at which the site for the retailer should be stored
func GetSitePath(retailerID string) string {
	return fmt.Sprintf("%s/%s/%s",
		common.RetailersCollection,
		retailerID,
		common.SitesCollection)
}

func GetSpokePath(retailerID string) string {
	return fmt.Sprintf("%s/%s/%s",
		common.RetailersCollection,
		retailerID,
		common.SpokesCollection)
}

func GetSiteSpokePath(retailerID string) string {
	return fmt.Sprintf("%s/%s/%s",
		common.RetailersCollection,
		retailerID,
		common.SiteSpokeCollection)
}

// IsValidEtagPresentInHeader will verify that header has valid etag value or not
// data is passed as interface so etag is calculated in this function.
// true if valid else false.
func IsValidEtagPresentInHeader(responseWriter http.ResponseWriter, request *http.Request, data interface{},
	logger *zap.SugaredLogger) bool {
	etag, err := GetETag(data)
	if err != nil {
		logger.Errorf("Error while getting etag for site data got from DB : %v", err)
		response.RespondWithInternalServerError(responseWriter, request)

		return false
	}
	//Check ETag
	if request.Header.Get(common.HeaderIfMatch) != etag {
		logger.Debugf("If-Match header value incorrect. ETag from DB is %s", etag)
		response.Respond(responseWriter, http.StatusPreconditionFailed,
			response.NewResponse(http.StatusPreconditionFailed,
				"If-Match header value incorrect, please get the latest and try again", nil),
			response.GetCommonResponseHeaders(request))

		return false
	}

	return true
}
