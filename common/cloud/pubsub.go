package cloud

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"github.com/TakeoffTech/site-info-svc/common"
	"github.com/TakeoffTech/site-info-svc/common/logging"
	"github.com/TakeoffTech/site-info-svc/common/utils"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"log"
	"os"
)

type PubSubRepository struct {
	client *pubsub.Client
	logger *zap.SugaredLogger
}

var PubSubRepositoryObj PubSubRepository

// NewPubSubRepository creates a PubSubRepositoryObj
func NewPubSubRepository(ctx context.Context) *PubSubRepository {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("pubsub.NewPubSubRepository"))
	defer span.End()
	if PubSubRepositoryObj.client != nil {
		PubSubRepositoryObj.logger = logging.GetLoggerFromContext(ctx)

		return &PubSubRepositoryObj
	}
	pubSubClient, err := pubsub.NewClient(ctx, os.Getenv(common.EnvProjectID))
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	PubSubRepositoryObj.client = pubSubClient
	PubSubRepositoryObj.logger = logging.GetLoggerFromContext(ctx)

	return &PubSubRepositoryObj
}

// Publish is a common method to publish any message to the topicName
func (p *PubSubRepository) Publish(ctx context.Context, topicName string, message any) {
	ctx, span := trace.StartSpan(ctx, utils.GetSpanName("pubsub.Publish"))
	defer span.End()
	topic := p.client.Topic(topicName)
	defer topic.Stop()
	var results []*pubsub.PublishResult

	bytes, err := json.Marshal(message)
	if err != nil {
		logging.GetLoggerFromContext(ctx).Errorf("Error while marshalling message to byte array: %v", err)

		return
	}

	r := topic.Publish(ctx, &pubsub.Message{
		Data: bytes,
	})

	results = append(results, r)

	for _, r := range results {
		id, err := r.Get(ctx)
		if err != nil {
			logging.GetLoggerFromContext(ctx).Errorf("Error while publishing message to audit pubsub: %v", err)

			return
		}
		logging.GetLoggerFromContext(ctx).Debugf("Message successfully published message id : %s", id)
	}
}
