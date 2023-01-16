package utils

import (
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetNextPageToken(t *testing.T) {
	t.Run("Get next page token from a valid key", func(t *testing.T) {
		got, err := GetNextPageToken("r12345", common.RetailersEncryptionKey)
		assert.NotEmpty(t, got)
		assert.Nil(t, err)
	})

	t.Run("Get next page token from a valid key for big data", func(t *testing.T) {
		got, err := GetNextPageToken("r123456789098765", common.RetailersEncryptionKey)
		assert.NotEmpty(t, got)
		assert.Nil(t, err)
	})

	t.Run("Failed to get next page token due to invalid key", func(t *testing.T) {
		got, err := GetNextPageToken("r12345", "invalid-key")
		assert.Empty(t, got)
		assert.NotNil(t, err)
	})
}

func TestDecodeNextPageToken(t *testing.T) {
	t.Run("Decode next page token from a valid key", func(t *testing.T) {
		got, err := DecodeNextPageToken("MTY2ODA3MDU2ODo6OjqiePc+UU1FVfcwyXX6EUpx", common.RetailersEncryptionKey)
		assert.NotEmpty(t, got)
		assert.Equal(t, "r123456789098765", got)
		assert.Nil(t, err)
	})

	t.Run("Failed to get next page token due to invalid key", func(t *testing.T) {
		got, err := DecodeNextPageToken("MTY2ODA6Ojo6RW5jb2Rl", "invalidity")
		assert.Empty(t, got)
		assert.NotNil(t, err)
	})

	t.Run("Failed to get next page token due to invalid token", func(t *testing.T) {
		got, err := DecodeNextPageToken("MTY2ODA6Ojo6RW5jb2Rlxsvsc", "invalidity")
		assert.Empty(t, got)
		assert.NotNil(t, err)
	})
}

func TestValidatePageToken(t *testing.T) {
	t.Run("Test with valid token", func(t *testing.T) {
		token, _ := GetNextPageToken("123346", common.RetailersEncryptionKey)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(common.HeaderPageToken, token)
		errs := ValidatePageToken(r, common.HeaderPageToken)
		assert.Nil(t, errs)
	})

	t.Run("Test with invalid token invalid base 64", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(common.HeaderPageToken, "MTY2ODA6Ojo6RW5jb2Rlxsvsc") //invalid base 64
		errs := ValidatePageToken(r, common.HeaderPageToken)
		assert.Equal(t, []string{"Invalid header value, unable to decrypt header : page_token"}, errs)
	})

	t.Run("Test with invalid token no colon separator in the token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(common.HeaderPageToken, "MTY2ODBFbmNvZGU=") //invalid token no separator
		errs := ValidatePageToken(r, common.HeaderPageToken)
		assert.Equal(t, []string{"Invalid page_token header : MTY2ODBFbmNvZGU="}, errs)
	})

	t.Run("Test with invalid token timestamp not present in before colon separator", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(common.HeaderPageToken, "aW52YWxpZGtleTo6OjoxNDc1MjM2OTgyMzYxMjU2")
		errs := ValidatePageToken(r, common.HeaderPageToken)
		assert.Equal(t, []string{"Invalid page_token : aW52YWxpZGtleTo6OjoxNDc1MjM2OTgyMzYxMjU2"}, errs)
	})

	t.Run("Test with invalid token expired token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(common.HeaderPageToken, "MTY2ODA3MDU2ODo6OjqiePc+UU1FVfcwyXX6EUpx")
		errs := ValidatePageToken(r, common.HeaderPageToken)
		assert.Equal(t, []string{"Header page_token expired : MTY2ODA3MDU2ODo6OjqiePc+UU1FVfcwyXX6EUpx"}, errs)
	})
}

func TestAddPaginationHeaderIfNotAdded(t *testing.T) {
	t.Run("Test with invalid token expired token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(common.HeaderPageToken, "MTY2ODA3MDU2ODo6OjqiePc+UU1FVfcwyXX6EUpx")
		headers := AddPaginationHeaderIfNotAdded(r)
		assert.Equal(t, headers[0], common.HeaderPageToken)
	})
}

func TestAddPaginationHeaderIfNotAdded_TwoHeaders(t *testing.T) {
	t.Run("Test with invalid token expired token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(common.HeaderPageToken, "MTY2ODA3MDU2ODo6OjqiePc+UU1FVfcwyXX6EUpx")
		r.Header.Set(common.HeaderPageSize, "93")
		headers := AddPaginationHeaderIfNotAdded(r)
		assert.Equal(t, headers[0], common.HeaderPageToken)
		assert.Equal(t, headers[1], common.HeaderPageSize)
	})
}

func TestGetPageSizeFromHeader(t *testing.T) {
	t.Run("Test with valid page size value", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		logger := logging.GetLoggerFromContext(r.Context())
		r.Header.Set(common.HeaderPageSize, "93")
		pageSize := GetPageSizeFromHeader(r, logger)
		assert.Equal(t, 93, pageSize)
	})
}
func TestGetPageSizeFromHeaderDefVal(t *testing.T) {
	t.Run("Test without page size header", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		logger := logging.GetLoggerFromContext(r.Context())
		pageSize := GetPageSizeFromHeader(r, logger)
		assert.Equal(t, common.DefaultPageSize, pageSize)
	})
}

func TestGetPageSizeFromHeaderInvalidVal(t *testing.T) {
	t.Run("Test with invalid page size value", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		logger := logging.GetLoggerFromContext(r.Context())
		r.Header.Set(common.HeaderPageSize, "HelloWorld")
		pageSize := GetPageSizeFromHeader(r, logger)
		assert.Equal(t, common.ReturnError, pageSize)
	})
}

func TestGetPageSizeFromHeaderInvalidIntVal(t *testing.T) {
	t.Run("Test with invalid integer page size value", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		logger := logging.GetLoggerFromContext(r.Context())
		r.Header.Set(common.HeaderPageSize, "1000")
		pageSize := GetPageSizeFromHeader(r, logger)
		assert.Equal(t, 1000, pageSize)
	})
}

func TestGetPageSizeFromHeaderInvalidFloatVal(t *testing.T) {
	t.Run("Test with invalid integer page size value", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		logger := logging.GetLoggerFromContext(r.Context())
		r.Header.Set(common.HeaderPageSize, "10.11")
		pageSize := GetPageSizeFromHeader(r, logger)
		assert.Equal(t, common.ReturnError, pageSize)
	})
}

func TestGetPageSizeFromHeaderEmptyVal(t *testing.T) {
	t.Run("Test with empty page size", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		logger := logging.GetLoggerFromContext(r.Context())
		r.Header.Set(common.HeaderPageSize, "")
		pageSize := GetPageSizeFromHeader(r, logger)
		assert.Equal(t, common.DefaultPageSize, pageSize)
	})
}
