package models

import (
	"time"
)

//nolint:lll
type Retailer struct {
	ID              string     `json:"id" validate:"disallowed" firestore:"id" structs:"id"`
	Name            string     `json:"name" validate:"required,name" firestore:"name" structs:"name"`
	CreatedBy       string     `json:"created_by" validate:"disallowed" firestore:"created_by" structs:"created_by"`
	UpdatedBy       string     `json:"updated_by" validate:"disallowed" firestore:"updated_by" structs:"updated_by"`
	DeactivatedBy   string     `json:"deactivated_by,omitempty" validate:"disallowed" firestore:"deactivated_by" structs:"deactivated_by"`
	CreatedTime     *time.Time `json:"created_time" validate:"disallowed" firestore:"created_time" structs:"created_time"`
	UpdatedTime     *time.Time `json:"updated_time" validate:"disallowed" firestore:"updated_time" structs:"updated_time"`
	DeactivatedTime *time.Time `json:"deactivated_time,omitempty" validate:"disallowed" firestore:"deactivated_time" structs:"deactivated_time"`
	ETag            string     `json:"etag,omitempty" validate:"disallowed" firestore:"-" structs:"-"`
}

type PubSubRetailerMessage struct {
	ChangeType string `json:"change_type"`
	RetailerID string `json:"retailer_id"`
}

func GetPubSubRetailerMessage(retailerID string, changeType string) *PubSubRetailerMessage {
	return &PubSubRetailerMessage{
		ChangeType: changeType,
		RetailerID: retailerID,
	}
}
