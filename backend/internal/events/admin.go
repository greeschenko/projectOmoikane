package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// EnsureTopic creates the named topic if it does not exist, and is idempotent
// (TopicAlreadyExists is tolerated). Partition count and replication factor
// align with the platform's single-node dev/CI deployment; managed clusters
// (MSK/Confluent) may pre-provision topics with their own settings and this
// call simply no-ops.
//
// Explicit provisioning is used instead of relying only on broker auto-create:
// auto-creation via metadata exchanges races the controller (first produce to a
// new topic can fail with UnknownTopicOrPartition), which breaks the outbox
// guarantee "relay publishes every pending row".
func EnsureTopic(ctx context.Context, brokers []string, topic string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("events: no kafka brokers configured")
	}
	if topic == "" {
		return nil
	}
	return EnsureTopics(ctx, brokers, []string{topic})
}

// EnsureTopics creates a list of topics (idempotent). This is what services
// call at startup (from their own provisioning run) BEFORE starting producer
// and consumer so the first publish can never race topic creation.
func EnsureTopics(ctx context.Context, brokers []string, topics []string) error {
	client := &kafka.Client{Addr: kafka.TCP(brokers...)}
	req := &kafka.CreateTopicsRequest{}
	for _, topic := range topics {
		if topic == "" {
			continue
		}
		req.Topics = append(req.Topics, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}
	if len(req.Topics) == 0 {
		return nil
	}
	res, err := client.CreateTopics(ctx, req)
	if err != nil {
		return fmt.Errorf("events: ensure topics: %w", err)
	}
	// Clients tolerates TopicAlreadyExists per topic, but treat any other
	// per-topic error as failure (clear message beats silent misdirection).
	for topic, terr := range res.Errors {
		if terr == nil || errors.Is(terr, kafka.TopicAlreadyExists) {
			continue
		}
		return fmt.Errorf("events: ensure topic %s: %w", topic, terr)
	}
	return nil
}
