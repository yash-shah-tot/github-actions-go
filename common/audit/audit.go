package audit

import (
	"fmt"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit/models"
	"time"
)

// PubSubAuditMessage is the structure of message that would be pushed to the pubsub topic
type PubSubAuditMessage struct {
	Path           string                 `json:"path"`
	ChangedBy      string                 `json:"changed_by"`
	EntityChanged  string                 `json:"entity_changed"`
	ChangeType     string                 `json:"change_type"`
	XCorrelationID string                 `json:"x_correlation_id"`
	OldEntity      map[string]interface{} `json:"old_entity"`
	NewEntity      map[string]interface{} `json:"new_entity"`
	ChangedAt      *time.Time             `json:"changed_at"`
	ExpiresAt      *time.Time             `json:"expires_at"`
}

func GetPubSubAuditMessage(path, xCorrelationID, changedBy,
	changeType, entityChanged string, changedAt *time.Time,
	oldEntity map[string]interface{}, newEntity map[string]interface{}) *PubSubAuditMessage {
	expiresAt := changedAt.Add(common.DataRetentionTime)

	return &PubSubAuditMessage{
		Path:           path,
		ChangedBy:      changedBy,
		EntityChanged:  entityChanged,
		ChangeType:     changeType,
		ChangedAt:      changedAt,
		XCorrelationID: xCorrelationID,
		OldEntity:      oldEntity,
		NewEntity:      newEntity,
		ExpiresAt:      &expiresAt,
	}
}

func NewAuditLog(changedBy string, changeType string,
	changeDetail []models.Diff, changedAt *time.Time, expiresAt *time.Time) *models.AuditLog {
	return &models.AuditLog{
		ChangedBy:     changedBy,
		ChangeType:    changeType,
		ChangeDetails: changeDetail,
		ChangedAt:     changedAt,
		ExpiresAt:     expiresAt,
	}
}

// GetRetailerAuditPath will return the firestore path at which the audit for the retailerID should be stored
func GetRetailerAuditPath(retailerID string) string {
	return fmt.Sprintf("%s/%s/%s",
		common.RetailersCollection,
		retailerID,
		common.RetailerAuditCollection)
}

// GetSiteAuditPath will return the firestore path at which the audit for the retailerID,siteID should be stored
func GetSiteAuditPath(retailerID, siteID string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		common.RetailersCollection,
		retailerID,
		common.SitesCollection,
		siteID,
		common.SiteAuditCollection)
}
