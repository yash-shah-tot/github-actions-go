package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	retailers "github.com/TakeoffTech/site-info-svc/cloud-functions/retailers/models"
	sites "github.com/TakeoffTech/site-info-svc/cloud-functions/sites/models"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/audit"
	"github.com/TakeoffTech/site-info-svc/common/audit/models"
	"github.com/TakeoffTech/site-info-svc/common/cloud"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/fatih/structs"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"reflect"
	"strings"
)

func init() {
	functions.CloudEvent("PushAudit", PushAudit)
}

type MessagePublishedData struct {
	Message PubSubMessage
}

type PubSubMessage struct {
	Data []byte `json:"data"`
}

func PushAudit(ctx context.Context, event event.Event) error {
	logger := logging.GetLoggerFromContext(ctx)
	var msg MessagePublishedData

	if err := event.DataAs(&msg); err != nil {
		logger.Errorf("Error occurred while converting event data to MessagePublishedData struct: %v", err)

		return err
	}

	return pushAuditLog(ctx, msg.Message.Data, cloud.NewFirestoreRepository(ctx))
}

func pushAuditLog(ctx context.Context, data []byte, dbClient cloud.DB) error {
	logger := logging.GetLoggerFromContext(ctx)
	var pubsubAuditMsg audit.PubSubAuditMessage
	err := json.Unmarshal(data, &pubsubAuditMsg)
	if err != nil {
		logger.Errorf("Error occurred while converting data to audit struct: %v", err)

		return err
	}
	auditLog := getAuditLog(&pubsubAuditMsg)
	logger = logging.GetLoggerWithXCorrelationID(pubsubAuditMsg.XCorrelationID)
	logger.Debugf("Audit Entity to be pushed %+v", auditLog)

	var retryCount int
	uniqueID := uuid.NewString()
	for retryCount = 0; retryCount < common.MaxRetryCount; retryCount++ {
		updateTime, err := dbClient.Save(ctx, pubsubAuditMsg.Path, uniqueID, auditLog)
		if err != nil {
			if status.Code(err) == codes.AlreadyExists {
				logger.Infof("Audit Log with ID %s already exists. Now generating new uuid for saving", uniqueID)
				uniqueID = uuid.NewString()

				continue
			} else {
				logger.Errorf("Error occurred while saving audit entity %v", err)
			}

			return err
		}
		logger.Infof("Audit Entity pushed successfully in %s at %v", pubsubAuditMsg.Path, updateTime)

		break
	}

	if retryCount == common.MaxRetryCount {
		logger.Errorf("Unable to create retailer after %d retires "+
			"as function was unable to generate unique id : %v", common.MaxRetryCount, err)

		return fmt.Errorf("unable to create retailer after %d retires as function was unable to generate unique id",
			common.MaxRetryCount)
	}

	return nil
}

// getAuditLog will compute the diff between old and new entity object and create an
// auditLog object which will be saved to the firestore
func getAuditLog(msg *audit.PubSubAuditMessage) *models.AuditLog {
	var diffs []models.Diff
	auditFields := getAuditChangeDetailFields(msg)
	switch msg.ChangeType {
	case common.AuditTypeCreate:
		for _, key := range auditFields {
			if msg.NewEntity[key] != nil {
				diffs = append(diffs, models.Diff{
					Field:    key,
					OldValue: nil,
					NewValue: msg.NewEntity[key],
				})
			}
		}
	case common.AuditTypeDeactivate:
		for _, key := range auditFields {
			if msg.OldEntity[key] != nil {
				diffs = append(diffs, models.Diff{
					Field:    key,
					OldValue: msg.OldEntity[key],
					NewValue: nil,
				})
			}
		}
	case common.AuditTypeUpdate:
		for _, key := range auditFields {
			if !reflect.DeepEqual(msg.OldEntity[key], msg.NewEntity[key]) {
				diffs = append(diffs, models.Diff{
					Field:    key,
					OldValue: msg.OldEntity[key],
					NewValue: msg.NewEntity[key],
				})
			}
		}
	}

	return audit.NewAuditLog(msg.ChangedBy, msg.ChangeType, diffs, msg.ChangedAt, msg.ExpiresAt)
}

// getAuditChangeDetailFields will return all the fields in the relevant entity changed struct
// which does not a disallowed value in the validate struct tag
func getAuditChangeDetailFields(msg *audit.PubSubAuditMessage) []string {
	var auditFields []string
	var fields []*structs.Field
	switch msg.EntityChanged {
	case common.EntityRetailer:
		fields = structs.Fields(retailers.Retailer{})
	case common.EntitySite:
		fields = structs.Fields(sites.Site{})
	default:
		return extractFields(msg.NewEntity)
	}
	for _, field := range fields {
		if !strings.Contains(field.Tag(common.Validate), common.Disallowed) {
			auditFields = append(auditFields, field.Tag(common.Firestore))
		}
	}

	return auditFields
}

func extractFields(entity map[string]interface{}) []string {
	var keys []string
	for key := range entity {
		keys = append(keys, key)
	}

	return keys
}
