package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer publishes CloudEvents to the platform events topic.
// Implementations must be safe for concurrent use (outbox relay batches).
type Producer interface {
	// Publish writes a CloudEvent envelope to the configured Topic.
	// It blocks until the broker acks (sync, RequireAll) so the outbox relay
	// can mark rows sent only after durable acknowledgment.
	Publish(ctx context.Context, ev CloudEvent) error
	Close() error
}

// KafkaProducer is the Kafka-backed Producer.
type KafkaProducer struct {
	writer *kafka.Writer
}

var _ Producer = (*KafkaProducer)(nil)

// NewProducer builds a Kafka-backed Producer with synchronous writes
// (RequiredAcks=RequireAll) so relays/flows get durability confirmation.
func NewProducer(cfg Config) (*KafkaProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("events: no kafka brokers configured")
	}
	if cfg.Topic == "" {
		cfg.Topic = DefaultTopic
	}
	// Managed-broker security (Phase 36): TLS+SASL transport when configured.
	if err := cfg.validateSecurity(); err != nil {
		return nil, err
	}
	tr, err := cfg.newTransport()
	if err != nil {
		return nil, err
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{}, // key = subject => same entity stays on one partition
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		Transport:    tr,
		// Small batch window: the relay publishes events in a loop and wants
		// low latency, not throughput batching (default is 1s per write).
		BatchTimeout:           50 * time.Millisecond,
		AllowAutoTopicCreation: true,
	}
	return &KafkaProducer{writer: w}, nil
}

// MarshalCloudEvent serializes an envelope to JSON bytes (shared by producer,
// outbox payloads, and tests).
func MarshalCloudEvent(ev CloudEvent) ([]byte, error) {
	return json.Marshal(ev)
}

// UnmarshalCloudEvent parses an envelope with CloudEvents validation
// (specversion must be 1.0; id/type/source required).
func UnmarshalCloudEvent(b []byte) (CloudEvent, error) {
	var ev CloudEvent
	if err := json.Unmarshal(b, &ev); err != nil {
		return ev, fmt.Errorf("events: invalid envelope json: %w", err)
	}
	if ev.SpecVersion != "1.0" || ev.ID == "" || ev.Type == "" || ev.Source == "" {
		return ev, fmt.Errorf("events: invalid envelope: missing specversion/id/type/source")
	}
	return ev, nil
}

// Publish implements Producer. The Kafka message key is the CloudEvent subject
// so events about one entity are ordered per partition; header carries the
// event type for cheap filtering by consumers.
func (p *KafkaProducer) Publish(ctx context.Context, ev CloudEvent) error {
	payload, err := MarshalCloudEvent(ev)
	if err != nil {
		return fmt.Errorf("events: marshal envelope: %w", err)
	}
	key := ev.Subject
	if key == "" {
		key = ev.Type
	}
	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "ce-type", Value: []byte(ev.Type)},
			{Key: "ce-source", Value: []byte(ev.Source)},
		},
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("events: publish %s: %w", ev.Type, err)
	}
	log.Printf("events: published %s (subject=%s)", ev.Type, key)
	return nil
}

// Close releases the underlying Kafka writer.
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
