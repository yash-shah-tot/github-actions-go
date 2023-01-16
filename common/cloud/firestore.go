package cloud

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"github.com/benpate/rosetta/convert"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"os"
	"time"
)

type FirestoreRepository struct {
	client *firestore.Client
	logger *zap.SugaredLogger
}

type Where struct {
	Field    string
	Operator string
	Value    interface{}
}

var FirestoreRepositoryObj FirestoreRepository

// NewFirestoreRepository creates a FirestoreRepositoryObj
func NewFirestoreRepository(ctx context.Context) *FirestoreRepository {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.NewFirestoreRepository"))
	defer span.End()
	if FirestoreRepositoryObj.client != nil {
		FirestoreRepositoryObj.logger = logging.GetLoggerFromContext(ctx)

		return &FirestoreRepositoryObj
	}
	firestoreClient, err := firestore.NewClient(ctx, os.Getenv(common.EnvProjectID))
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	FirestoreRepositoryObj.client = firestoreClient
	FirestoreRepositoryObj.logger = logging.GetLoggerFromContext(ctx)

	return &FirestoreRepositoryObj
}

// Exists function is used to check if the field==value in the collectionID passed
// will return true if the field==value else false
func (f *FirestoreRepository) Exists(ctx context.Context,
	collectionPath string, field string, value string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.Exists"))
	defer span.End()
	docItr := f.client.Collection(collectionPath).Where(field, common.OperatorEquals, value).Documents(ctx)
	for {
		doc, err := docItr.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			f.logger.Errorf("Error occurred while checking existence of document in DB: %v", err)

			return false, err
		}
		if doc.Exists() {
			return true, nil
		}
	}

	return false, nil
}

// Save function will save the document in the DB with the given collectionID and documentID
func (f *FirestoreRepository) Save(ctx context.Context,
	collectionPath string, documentID string, document interface{}) (time.Time, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.Save"))
	defer span.End()
	result, err := f.client.Collection(collectionPath).Doc(documentID).Create(ctx, document)
	if err != nil {
		f.logger.Errorf("Error occurred while saving the document to DB : %v", err)

		return time.Time{}, err
	}

	return result.UpdateTime, nil
}

// GetByID returns a single document for the passed collectionID and documentID
func (f *FirestoreRepository) GetByID(ctx context.Context,
	collectionPath string, documentID string, skipDeactivated bool) (map[string]interface{}, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.GetByID"))
	defer span.End()
	var query firestore.Query
	query = f.client.Collection(collectionPath).Where(common.ID, common.OperatorEquals, documentID)
	if skipDeactivated {
		query = query.Where(common.DeactivatedTime, common.OperatorEquals, nil)
	}
	result, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(result) > 1 {
		err = status.Error(codes.Internal, "Multiple documents with same ID")
		f.logger.Debugf("multiple documents with same ID: %s present", documentID)

		return nil, err
	}
	if len(result) == 0 && err == nil {
		err = status.Error(codes.NotFound, "document not found")
		f.logger.Debugf("Document ID %s not found", documentID)

		return nil, err
	}

	return result[0].Data(), nil
}

// Update will perform all the updates passed in updates []firestore.Update for the collectionID and documentID
func (f *FirestoreRepository) Update(ctx context.Context,
	collectionPath string, documentID string, updates []firestore.Update) (time.Time, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.Update"))
	defer span.End()
	result, err := f.client.Collection(collectionPath).Doc(documentID).Update(ctx, updates)
	if err != nil {
		f.logger.Errorf("Error occurred while updating the document to DB : %v", err)

		return result.UpdateTime, err
	}

	return result.UpdateTime, nil
}

// GetAll will return all documents under the collectionID with deleted param
// if skipDeactivated is passed true then where clause of deactivated_time==nil will be added to the query
func (f *FirestoreRepository) GetAll(ctx context.Context,
	collectionPath string, pageDetails Page, whereClauses []Where) ([]map[string]interface{}, string, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.GetAll"))
	defer span.End()
	var result []map[string]interface{}
	collection := f.client.Collection(collectionPath)
	var query firestore.Query
	var zeroStartAfterID bool
	//This switch case has been added so that we come to know what type of object we are dealing with
	//Check if the startAfterID is zero value which means we don't have to add startAfter in the query
	//if the startAfterID is non-zero then it's added to startAfter in the query
	switch startAfterID := pageDetails.StartAfterID.(type) {
	case time.Time:
		zeroStartAfterID = startAfterID.IsZero()
	default:
		zeroStartAfterID = convert.IsZeroValue(startAfterID)
	}
	if !zeroStartAfterID {
		query = collection.OrderBy(pageDetails.OrderBy, pageDetails.Sort).
			StartAfter(pageDetails.StartAfterID).
			Limit(pageDetails.PageSize)
	} else {
		query = collection.OrderBy(pageDetails.OrderBy, pageDetails.Sort).Limit(pageDetails.PageSize)
	}

	//Add all where clauses to the query
	for _, where := range whereClauses {
		query = query.Where(where.Field, where.Operator, where.Value)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		f.logger.Errorf("Error occurred while fetching the documents from DB : %v", err)

		return nil, "", err
	}

	var lastDocID string
	if len(docs) > 0 {
		lastDocID = convert.StringDefault(docs[len(docs)-1].Data()[pageDetails.OrderBy], "")
	}

	for _, doc := range docs {
		result = append(result, doc.Data())
	}

	return result, lastDocID, nil
}

// ExistsInCollectionGroup function is used to check if the field==value in the collectionGroup passed
// will return true if the field==value else false
func (f *FirestoreRepository) ExistsInCollectionGroup(ctx context.Context,
	collectionGroupID string, field string, value string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.ExistsInCollectionGroup"))
	defer span.End()
	docItr := f.client.CollectionGroup(collectionGroupID).Where(field, common.OperatorEquals, value).Documents(ctx)
	for {
		doc, err := docItr.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			f.logger.Errorf("Error occurred while checking existence of document in DB: %v", err)

			return false, err
		}
		if doc.Exists() {
			return true, nil
		}
	}

	return false, nil
}

// CheckSubDocuments function checks if the parent document is deletable
// by checking if all its subDocuments are deleted(if any)
// Returns a bool value true if all its subDocuments are deleted
func (f *FirestoreRepository) CheckSubDocuments(ctx context.Context, collectionPath string,
	documentID string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.CheckSubDocuments"))
	defer span.End()
	result, err := f.client.Collection(collectionPath).
		Where(common.DeactivatedTime, common.OperatorEquals, nil).
		Documents(ctx).
		GetAll()
	if err != nil {
		return false, err
	} else if result == nil && err == nil {
		return true, nil
	}

	return false, nil
}

// Delete function will delete the document id provided under the collection path
// This will be used only for collection site-info-site-spoke collection
func (f *FirestoreRepository) Delete(ctx context.Context, collectionPath string,
	documentID string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("firestore.Delete"))
	defer span.End()
	_, err := f.client.Collection(collectionPath).Doc(documentID).Delete(ctx)
	if err != nil {
		f.logger.Errorf("Error occurred while deleting the document to DB : %v", err)

		return false, err
	}

	return true, nil
}
