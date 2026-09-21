package events

import (
	"os"
	"strings"
	"time"
)

// Config holds the Kafka + outbox-relay settings shared by every service.
// Topics are shared platform constants so producers and consumers always agree
// (services configure only brokers + their own consumer group).
type Config struct {
	// Brokers is the Kafka bootstrap list, e.g. ["kafka:9092"] in compose,
	// ["localhost:9092"] outside Docker.
	Brokers []string

	// Topic is where all CloudEvents are published (single backbone topic).
	Topic string

	// DLQTopic receives events whose handler failed after ConsumerMaxRetries.
	DLQTopic string

	// ConsumerMaxRetries is how many times a consumer retries a handler before
	// moving the event to the DLQ (attempts = 1 initial + this many retries).
	ConsumerMaxRetries int

	// RetryBackoff is the delay between consumer handler retries.
	RetryBackoff time.Duration

	// RelayBatchSize is how many pending outbox rows the relay publishes per run.
	RelayBatchSize int

	// RelayInterval is how often the relay polls the outbox.
	RelayInterval time.Duration

	// OutboxMaxAttempts is how many publish attempts the relay makes before
	// marking an outbox row failed (these surface for operator attention and
	// are NOT dropped).
	OutboxMaxAttempts int
}

const (
	// DefaultTopic carries every CloudEvent on the platform.
	DefaultTopic = "omoikane.events"
	// DefaultDLQTopic receives events that exhausted consumer retries.
	DefaultDLQTopic = "omoikane.events.dlq"
)

// DefaultConfig returns sane single-broker defaults (compose semantics:
// broker reachable at "kafka:9092" from services, "localhost:9092" from host).
func DefaultConfig() Config {
	return Config{
		Brokers:            []string{"kafka:9092"},
		Topic:              DefaultTopic,
		DLQTopic:           DefaultDLQTopic,
		ConsumerMaxRetries: 3,
		RetryBackoff:       200 * time.Millisecond,
		RelayBatchSize:     100,
		RelayInterval:      time.Second,
		OutboxMaxAttempts:  5,
	}
}

// ConfigFromEnv builds a Config from the standard KAFKA_* environment
// variables. Overrides DefaultConfig so tests and local runs can point
// anywhere (e.g. KAFKA_BROKERS=localhost:9092).
func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		cfg.Brokers = strings.Split(v, ",")
	}
	if v := os.Getenv("KAFKA_EVENTS_TOPIC"); v != "" {
		cfg.Topic = v
	}
	if v := os.Getenv("KAFKA_DLQ_TOPIC"); v != "" {
		cfg.DLQTopic = v
	}
	return cfg
}
