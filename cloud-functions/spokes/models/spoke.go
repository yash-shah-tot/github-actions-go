package models

import (
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/models"
	"time"
)

//nolint:lll
type Spoke struct {
	ID              string           `json:"id" validate:"disallowed" firestore:"id" structs:"id"`
	Name            string           `json:"name" validate:"required,name" firestore:"name" structs:"name"`
	RetailerID      string           `json:"retailer_id" validate:"disallowed" firestore:"retailer_id" structs:"retailer_id"`
	Timezone        string           `json:"timezone" validate:"disallowed" firestore:"timezone" structs:"timezone"`
	Location        *models.Location `json:"location" validate:"required" firestore:"location" structs:"location"`
	CreatedBy       string           `json:"created_by" validate:"disallowed" firestore:"created_by" structs:"created_by"`
	UpdatedBy       string           `json:"updated_by" validate:"disallowed" firestore:"updated_by" structs:"updated_by"`
	DeactivatedBy   string           `json:"deactivated_by,omitempty" validate:"disallowed" firestore:"deactivated_by" structs:"deactivated_by"`
	CreatedTime     *time.Time       `json:"created_time" validate:"disallowed" firestore:"created_time" structs:"created_time"`
	UpdatedTime     *time.Time       `json:"updated_time" validate:"disallowed" firestore:"updated_time" structs:"updated_time"`
	DeactivatedTime *time.Time       `json:"deactivated_time,omitempty" validate:"disallowed" firestore:"deactivated_time" structs:"deactivated_time"`
	ETag            string           `json:"etag,omitempty" validate:"disallowed" firestore:"-" structs:"-"`
}

// SiteSpoke site spoke struct
//
//nolint:lll
type SiteSpoke struct {
	ID          string     `json:"id" validate:"disallowed" firestore:"id" structs:"id"`
	SiteID      string     `json:"site_id" validate:"disallowed" firestore:"site_id" structs:"site_id"`
	SpokeID     string     `json:"spoke_id" validate:"disallowed" firestore:"spoke_id" structs:"spoke_id"`
	RetailerID  string     `json:"retailer_id" validate:"disallowed" firestore:"retailer_id" structs:"retailer_id"`
	CreatedBy   string     `json:"created_by" validate:"disallowed" firestore:"created_by" structs:"created_by"`
	CreatedTime *time.Time `json:"created_time" validate:"disallowed" firestore:"created_time" structs:"created_time"`
}

func NewSiteSpoke(siteID string, spokeID string, retailerID string, createdBy string) SiteSpoke {
	currentTime := time.Now().UTC().Round(time.Second)

	return SiteSpoke{
		ID:          GetSiteSpokeID(siteID, spokeID),
		SiteID:      siteID,
		SpokeID:     spokeID,
		RetailerID:  retailerID,
		CreatedBy:   createdBy,
		CreatedTime: &currentTime,
	}
}

// IsValidLocationData is used to check if location data is valid for site.
// true is data is valid, else false
func (spoke Spoke) IsValidLocationData() bool {
	if spoke.Location == nil || spoke.Location.Longitude == nil || spoke.Location.Latitude == nil {
		return false
	}

	return true
}

func GetRequiredHeaders() []string {
	requiredHeaders := common.GetMandatoryHeaders()
	requiredHeaders = append(requiredHeaders, common.HeaderRetailerID)

	return requiredHeaders
}

type PubSubSpokeMessage struct {
	ChangeType  string `json:"change_type"`
	RetailerID  string `json:"retailer_id"`
	SiteID      string `json:"site_id"`
	SpokeID     string `json:"spoke_id"`
	SiteSpokeID string `json:"id"`
}

func GetPubSubSpokeMessage(retailerID string, siteID string, spokeID string,
	siteSpokeID string, changeType string) *PubSubSpokeMessage {
	return &PubSubSpokeMessage{
		ChangeType:  changeType,
		RetailerID:  retailerID,
		SiteID:      siteID,
		SpokeID:     spokeID,
		SiteSpokeID: siteSpokeID,
	}
}

func GetSiteSpokeID(siteID string, spokeID string) string {
	return fmt.Sprintf("%s_%s", siteID, spokeID)
}
