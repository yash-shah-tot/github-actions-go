package models

import (
	"time"
)

// AuditLog is the structure of audit document that would be created for entity audits
type AuditLog struct {
	ChangedBy     string     `json:"changed_by" firestore:"changed_by"`
	ChangeType    string     `json:"change_type" firestore:"change_type"`
	ChangeDetails []Diff     `json:"change_details" firestore:"change_details"`
	ChangedAt     *time.Time `json:"changed_at" firestore:"changed_at"`
	ExpiresAt     *time.Time `json:"-" firestore:"expires_at"`
}

// Diff struct to specify what was the old value of the field and the new value
type Diff struct {
	Field    string      `json:"field" firestore:"field"`
	OldValue interface{} `json:"old_value" firestore:"old_value"`
	NewValue interface{} `json:"new_value" firestore:"new_value"`
}
