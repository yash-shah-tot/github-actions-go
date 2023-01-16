package utils

import (
	"context"
	"errors"
	"github.com/TakeoffTech/site-info-svc/cloud-functions/retailers/models"
	"github.com/TakeoffTech/site-info-svc/common"
	commonModels "github.com/TakeoffTech/site-info-svc/common/models"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func init() {
	err := os.Setenv(common.GoogleMapsAPIEnv, "api-key")
	if err != nil {
		return
	}
}
func TestComputeEtag(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"ETag for Retailer Object",
			args{data: []byte{123, 34, 110, 97, 109, 101, 34, 58, 34, 109, 101, 34, 44, 34, 112, 104, 111, 110, 101, 34, 58, 34, 49, 50, 51, 52, 53, 34, 125}},
			"9af73e130d09e5768f223cf690ca38dfa1d6e1e42091bcb7a2e40c667588493f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeEtag(tt.args.data); got != tt.want {
				t.Errorf("GetEtag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToObject(t *testing.T) {
	type args struct {
		data   map[string]interface{}
		object interface{}
	}

	type testCase struct {
		name    string
		args    args
		wantErr bool
	}

	tests := []testCase{
		{
			"Error while marshalling ",
			args{
				map[string]interface{}{
					"foo": make(chan int),
				},
				&models.Retailer{},
			},
			true,
		},
		{
			"Success conversion of object ",
			args{
				map[string]interface{}{},
				&models.Retailer{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ConvertToObject(tt.args.data, tt.args.object); (err != nil) != tt.wantErr {
				t.Errorf("ConvertToObject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	type args struct {
		bytes  []byte
		object interface{}
	}
	type testCase struct {
		name    string
		args    args
		wantErr bool
	}

	tests := []testCase{
		{
			"Error while Umarshal",
			args{
				[]byte{},
				"string",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := unmarshal(tt.args.bytes, tt.args.object); (err != nil) != tt.wantErr {
				t.Errorf("Umarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetSpanName(t *testing.T) {
	type args struct {
		spanName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Test Span Name",
			args{
				"GetRetailer",
			},
			common.ServiceName + "." + "GetRetailer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetSpanName(tt.args.spanName), "GetSpanName(%v)", tt.args.spanName)
		})
	}
}

func TestGetETag(t *testing.T) {
	t.Run("Get Etag from Map", func(t *testing.T) {
		etag, err := GetETag(map[string]interface{}{"id": "r12345"})
		assert.Nil(t, err)
		assert.Equal(t, "502250b1d731426e35876ff70a5ae911fe08e69e0146417fa43df4b316ca62e9", etag)
	})

	t.Run("Get Etag from invalid map", func(t *testing.T) {
		etag, err := GetETag(map[string]interface{}{"id": make(chan int)})
		assert.NotNil(t, err)
		assert.Equal(t, "", etag)
	})

	t.Run("Get Etag from invalid data type", func(t *testing.T) {
		etag, err := GetETag("abc")
		assert.NotNil(t, err)
		assert.Equal(t, errors.New("type of data did not match map or struct, returning empty etag"), err)
		assert.Equal(t, "", etag)
	})

	t.Run("Get Etag from struct", func(t *testing.T) {
		etag, err := GetETag(models.Retailer{})
		assert.Nil(t, err)
		assert.Equal(t, "458224b4d054192db04833dfa63e6ea94e833b8744bef759b8b66299f4d6c2b8", etag)
	})
}

func TestPopulateETags(t *testing.T) {
	t.Run("Populate Etag for list of Map", func(t *testing.T) {
		var list []map[string]interface{}
		err := PopulateETags([]map[string]interface{}{{"id": "r12345"}, {"id": "r12345"}, {"id": "r98765"}}, &list)
		assert.Nil(t, err)
		assert.Equal(t, "502250b1d731426e35876ff70a5ae911fe08e69e0146417fa43df4b316ca62e9", list[0]["etag"])
		assert.Equal(t, "502250b1d731426e35876ff70a5ae911fe08e69e0146417fa43df4b316ca62e9", list[1]["etag"])
		assert.Equal(t, "baac12b08a36d2d0761abfbe0709cd675fa5d522d29e2991f582ce0591ff2470", list[2]["etag"])
	})

	t.Run("Populate Etag for list of Map invalid map", func(t *testing.T) {
		var list []map[string]interface{}
		err := PopulateETags([]map[string]interface{}{{"id": "r12345"}, {"id": make(chan int)}, {"id": "r98765"}}, &list)
		assert.NotNil(t, err)
	})
}

func TestGetTimeZone(t *testing.T) {
	t.Run("Get Time Zone for correct values of locations", func(t *testing.T) {
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
		TimeZone, err := GetTimeZone(context.Background(), 54.25, 13.134)
		assert.Nil(t, err)
		assert.Equal(t, "Europe/Berlin", TimeZone.TimezoneID)
	})

	t.Run("Get Time Zone for out of range values of locations", func(t *testing.T) {
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
		_, err := GetTimeZone(context.Background(), 190.25, 13.134)
		assert.Equal(t, "error : google Status : INVALID_REQUEST", err.Error())
	})
}

func TestGetSitePath(t *testing.T) {
	t.Run("Get correct path", func(t *testing.T) {
		path := GetSitePath("r12345")
		assert.Equal(t, "site-info-retailers/r12345/site-info-sites", path)
	})
}

func TestGetSpokePath(t *testing.T) {
	t.Run("Get correct path", func(t *testing.T) {
		path := GetSpokePath("r12345")
		assert.Equal(t, "site-info-retailers/r12345/site-info-spokes", path)
	})
}

func TestGetSiteSpokePath(t *testing.T) {
	t.Run("Get correct path", func(t *testing.T) {
		path := GetSiteSpokePath("r12345")
		assert.Equal(t, "site-info-retailers/r12345/site-info-site-spoke", path)
	})
}
