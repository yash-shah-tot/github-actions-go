package audit

import (
	"context"
	"encoding/json"
	"errors"
	retailer "github.com/TakeoffTech/site-info-svc/cloud-functions/retailers/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	"github.com/TakeoffTech/site-info-svc/common/audit/models"
	"github.com/TakeoffTech/site-info-svc/mocks"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/fatih/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"testing"
	"time"
)

func init() {
	err := os.Setenv(common.EnvProjectID, "project-id")
	if err != nil {
		return
	}
}

func Test_getAuditChangeDetailFields(t *testing.T) {
	t.Run("Get Audit Fields for entity Retailer", func(t *testing.T) {
		msg := &audit.PubSubAuditMessage{EntityChanged: common.EntityRetailer}
		fields := getAuditChangeDetailFields(msg)
		assert.Equal(t, []string{"name"}, fields)
	})
	t.Run("Get Audit Fields when entity not matched", func(t *testing.T) {
		msg := &audit.PubSubAuditMessage{EntityChanged: "invalid"}
		fields := getAuditChangeDetailFields(msg)
		assert.Equal(t, 0, len(fields))
	})
}

func Test_getAuditEntity(t *testing.T) {
	t.Run("Get entity audit for retailer create", func(t *testing.T) {
		currentTime := time.Now()
		expires := currentTime.Add(common.DataRetentionTime)
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeCreate, common.EntityRetailer, &currentTime,
			nil, structs.Map(retailer.Retailer{Name: "RetailerName"}))
		auditLog := getAuditLog(msg)
		assert.Equal(t, "user", auditLog.ChangedBy)
		assert.Equal(t, "create", auditLog.ChangeType)
		assert.Equal(t, &currentTime, auditLog.ChangedAt)
		assert.Equal(t, &expires, auditLog.ExpiresAt)
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{NewValue: "RetailerName", Field: "name"})
	})

	t.Run("Get entity audit for site create", func(t *testing.T) {
		currentTime := time.Now()
		expires := currentTime.Add(common.DataRetentionTime)
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "Site Name",
			"retailer_site_id": "R Site ID",
			"retailer_id":      "r12345",
			"location": map[string]interface{}{
				"lat":  50,
				"long": 50,
			},
		}
		var msg = audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeCreate, common.EntitySite, &currentTime, nil, site)
		auditLog := getAuditLog(msg)
		assert.Equal(t, "user", auditLog.ChangedBy)
		assert.Equal(t, "create", auditLog.ChangeType)
		assert.Equal(t, &currentTime, auditLog.ChangedAt)
		assert.Equal(t, &expires, auditLog.ExpiresAt)
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{NewValue: "Site Name", Field: "name"})
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{NewValue: "R Site ID", Field: "retailer_site_id"})
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{NewValue: map[string]interface{}{"lat": 50, "long": 50}, Field: "location"})
	})

	t.Run("Get entity audit for retailer update", func(t *testing.T) {
		currentTime := time.Now()
		expires := currentTime.Add(common.DataRetentionTime)
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeUpdate, common.EntityRetailer, &currentTime,
			structs.Map(retailer.Retailer{Name: "OldRetailerName"}), structs.Map(retailer.Retailer{Name: "NewRetailerName"}))
		auditLog := getAuditLog(msg)
		assert.Equal(t, "user", auditLog.ChangedBy)
		assert.Equal(t, "update", auditLog.ChangeType)
		assert.Equal(t, &currentTime, auditLog.ChangedAt)
		assert.Equal(t, &expires, auditLog.ExpiresAt)
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: "OldRetailerName", NewValue: "NewRetailerName", Field: "name"})
	})

	t.Run("Get entity audit for site update", func(t *testing.T) {
		currentTime := time.Now()
		expires := currentTime.Add(common.DataRetentionTime)
		oldSite := map[string]interface{}{
			"id":               "s12345",
			"name":             "Site Name",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"location": map[string]interface{}{
				"lat":  50,
				"long": 50,
			},
		}
		newSite := map[string]interface{}{
			"id":               "s12345",
			"name":             "Site Name Update",
			"retailer_site_id": "ABC123",
			"retailer_id":      "r12345",
			"location": map[string]interface{}{
				"lat":  10,
				"long": 10,
			},
		}
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeUpdate, common.EntitySite, &currentTime,
			oldSite, newSite)
		auditLog := getAuditLog(msg)
		assert.Equal(t, "user", auditLog.ChangedBy)
		assert.Equal(t, "update", auditLog.ChangeType)
		assert.Equal(t, &currentTime, auditLog.ChangedAt)
		assert.Equal(t, &expires, auditLog.ExpiresAt)
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: "Site Name", Field: "name", NewValue: "Site Name Update"})
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: map[string]interface{}{"lat": 50, "long": 50},
			NewValue: map[string]interface{}{"lat": 10, "long": 10},
			Field:    "location"})
	})

	t.Run("Get entity audit for site status update", func(t *testing.T) {
		currentTime := time.Now()
		expires := currentTime.Add(common.DataRetentionTime)
		oldSite := map[string]interface{}{
			"status": "active",
		}
		newSite := map[string]interface{}{
			"status": "inactive",
		}
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeUpdate, common.Status, &currentTime,
			oldSite, newSite)
		auditLog := getAuditLog(msg)
		assert.Equal(t, "user", auditLog.ChangedBy)
		assert.Equal(t, "update", auditLog.ChangeType)
		assert.Equal(t, &currentTime, auditLog.ChangedAt)
		assert.Equal(t, &expires, auditLog.ExpiresAt)
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: "active", Field: "status", NewValue: "inactive"})
	})

	t.Run("Get entity audit for retailer delete", func(t *testing.T) {
		currentTime := time.Now()
		expires := currentTime.Add(common.DataRetentionTime)
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeDeactivate, common.EntityRetailer, &currentTime,
			structs.Map(retailer.Retailer{Name: "OldRetailerName"}), nil)
		auditLog := getAuditLog(msg)
		assert.Equal(t, "user", auditLog.ChangedBy)
		assert.Equal(t, "deactivate", auditLog.ChangeType)
		assert.Equal(t, &currentTime, auditLog.ChangedAt)
		assert.Equal(t, &expires, auditLog.ExpiresAt)
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: "OldRetailerName", Field: "name"})
	})

	t.Run("Get entity audit for site deactivate", func(t *testing.T) {
		currentTime := time.Now()
		expires := currentTime.Add(common.DataRetentionTime)
		site := map[string]interface{}{
			"id":               "s12345",
			"name":             "Site Name",
			"retailer_site_id": "R Site ID",
			"retailer_id":      "r12345",
			"location": map[string]interface{}{
				"lat":  50,
				"long": 50,
			},
		}
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeDeactivate, common.EntitySite, &currentTime,
			site, nil)
		auditLog := getAuditLog(msg)
		assert.Equal(t, "user", auditLog.ChangedBy)
		assert.Equal(t, "deactivate", auditLog.ChangeType)
		assert.Equal(t, &currentTime, auditLog.ChangedAt)
		assert.Equal(t, &expires, auditLog.ExpiresAt)
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: "Site Name", Field: "name"})
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: "R Site ID", Field: "retailer_site_id"})
		assert.Contains(t, auditLog.ChangeDetails, models.Diff{OldValue: map[string]interface{}{"lat": 50, "long": 50},
			Field: "location"})
	})
}

func Test_pushAuditEntity(t *testing.T) {
	t.Run("Push Audit Entity Successfully", func(t *testing.T) {
		firestore := mocks.NewDB(t)
		currentTime := time.Now().UTC().Round(time.Second)
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeCreate, common.EntityRetailer, &currentTime,
			nil, structs.Map(retailer.Retailer{Name: "RetailerName"}))
		bytes, _ := json.Marshal(msg)
		auditLog := getAuditLog(msg)
		firestore.On("Save", mock.Anything, msg.Path, mock.Anything, auditLog).Return(time.Now(), nil)
		err := pushAuditLog(context.Background(), bytes, firestore)
		assert.Nil(t, err)
	})

	t.Run("Push Audit Entity Failed with connection timeout", func(t *testing.T) {
		firestore := mocks.NewDB(t)
		currentTime := time.Now().UTC().Round(time.Second)
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeCreate, common.EntityRetailer, &currentTime,
			nil, structs.Map(retailer.Retailer{Name: "RetailerName"}))
		bytes, _ := json.Marshal(msg)
		auditLog := getAuditLog(msg)
		firestore.On("Save", mock.Anything, msg.Path, mock.Anything, auditLog).Return(currentTime, errors.New("connection timeout"))
		err := pushAuditLog(context.Background(), bytes, firestore)
		assert.NotNil(t, err)
	})

	t.Run("Push Audit Entity Failed with already exist error after retries", func(t *testing.T) {
		firestore := mocks.NewDB(t)
		currentTime := time.Now().UTC().Round(time.Second)
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeCreate, common.EntityRetailer, &currentTime,
			nil, structs.Map(retailer.Retailer{Name: "RetailerName"}))
		bytes, _ := json.Marshal(msg)
		firestore.On("Save", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(currentTime, status.Error(codes.AlreadyExists, "document id already exist"))
		err := pushAuditLog(context.Background(), bytes, firestore)
		assert.NotNil(t, err)
	})

	t.Run("Push Audit Entity Failed with already exist success after 1 retry", func(t *testing.T) {
		firestore := mocks.NewDB(t)
		currentTime := time.Now().UTC().Round(time.Second)
		msg := audit.GetPubSubAuditMessage("path", "123", "user",
			common.AuditTypeCreate, common.EntityRetailer, &currentTime,
			nil, structs.Map(retailer.Retailer{Name: "RetailerName"}))
		bytes, _ := json.Marshal(msg)
		firestore.On("Save", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(currentTime, status.Error(codes.AlreadyExists, "document id already exist")).Once()
		firestore.On("Save", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(currentTime, nil).Once()
		err := pushAuditLog(context.Background(), bytes, firestore)
		assert.Nil(t, err)
	})

	t.Run("Bad data bytes passed", func(t *testing.T) {
		firestore := mocks.NewDB(t)
		err := pushAuditLog(context.Background(), nil, firestore)
		assert.NotNil(t, err)
	})
}

func TestPushAudit(t *testing.T) {
	t.Run("Invalid event Received", func(t *testing.T) {
		e := event.New()
		e.DataEncoded = []byte("invalid")
		e.SetDataContentType("invalid")
		err := PushAudit(context.Background(), e)
		assert.NotNil(t, err)
	})
}
