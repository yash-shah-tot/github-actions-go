package cloud

import (
	"cloud.google.com/go/firestore"
	"context"
	"time"
)

// DB interface
// This interface is a common interface for the DB operations
type DB interface {
	GetAll(ctx context.Context, collectionPath string, pageDetails Page,
		whereClauses []Where) ([]map[string]interface{}, string, error)
	GetByID(ctx context.Context, collectionPath string, documentID string,
		skipDeactivated bool) (map[string]interface{}, error)
	Exists(ctx context.Context, collectionPath string, field string, value string) (bool, error)
	ExistsInCollectionGroup(ctx context.Context, collectionGroupID string, field string, value string) (bool, error)
	Save(ctx context.Context,
		collectionPath string, documentID string, document interface{}) (time.Time, error)
	Update(ctx context.Context,
		collectionPath string, documentID string, document []firestore.Update) (time.Time, error)
	CheckSubDocuments(ctx context.Context, collectionPath string, documentID string) (bool, error)
	Delete(ctx context.Context, collectionPath string, documentID string) (bool, error)
}

// Queue interface
// This interface is a common interface for the Queue/PubSub operations
type Queue interface {
	Publish(ctx context.Context, topicName string, response any)
}

type Page struct {
	StartAfterID any
	PageSize     int
	OrderBy      string
	Sort         firestore.Direction
}
