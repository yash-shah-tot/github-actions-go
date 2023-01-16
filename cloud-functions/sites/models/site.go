package models

import (
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/models"
	"time"
)

//nolint:lll
type Site struct {
	ID              string           `json:"id" validate:"disallowed" firestore:"id" structs:"id"`
	Name            string           `json:"name" validate:"required,name" firestore:"name" structs:"name"`
	RetailerSiteID  string           `json:"retailer_site_id" validate:"required" firestore:"retailer_site_id" structs:"retailer_site_id"`
	RetailerID      string           `json:"retailer_id" validate:"disallowed" firestore:"retailer_id" structs:"retailer_id"`
	Status          string           `json:"status" validate:"disallowed" firestore:"status" structs:"status"`
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

type SiteStatuses struct {
	ID                string              `json:"id"`
	StatusTransitions map[string][]string `json:"status-transitions"`
	ExpiresAt         time.Time           `json:"-"`
}

// IsValidLocationData is used to check if location data is valid for site.
// true is data is valid, else false
func (site Site) IsValidLocationData() bool {
	if site.Location == nil || site.Location.Longitude == nil || site.Location.Latitude == nil {
		return false
	}

	return true
}

func GetRequiredHeaders() []string {
	requiredHeaders := common.GetMandatoryHeaders()
	requiredHeaders = append(requiredHeaders, common.HeaderRetailerID)

	return requiredHeaders
}

type PubSubSiteMessage struct {
	ChangeType string `json:"change_type"`
	RetailerID string `json:"retailer_id"`
	SiteID     string `json:"site_id"`
}

func GetPubSubSiteMessage(retailerID string, siteID string, changeType string) *PubSubSiteMessage {
	return &PubSubSiteMessage{
		ChangeType: changeType,
		RetailerID: retailerID,
		SiteID:     siteID,
	}
}
