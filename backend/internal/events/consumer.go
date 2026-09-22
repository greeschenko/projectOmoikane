package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Handler processes a single CloudEvent. It should be idempotent: consumers
// use at-least-once semantics (the offset is committed only after success or
// DLQ dispatch), so a handler may see the same event more than once after a
// crash or retry.
type Handler interface {
	Handle(ctx context.Context, ev CloudEvent) error
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(ctx context.Context, ev CloudEvent) error

// Handle implements Handler.
func (f HandlerFunc) Handle(ctx context.Context, ev CloudEvent) error { return f(ctx, ev) }

// Consumer subscribes to the platform events topic as a consumer GROUP member
// and dispatches each CloudEvent to a Handler. Behavior:
//   - at-least-once: the offset is committed after the message is handled
//     successfully OR moved to the DLQ — a crash between fetch and commit
//     redelivers (handlers must be idempotent).
//   - retries: handler errors are retried ConsumerMaxRetries times with
//     RetryBackoff between attempts.
//   - DLQ: after retries are exhausted the raw message (with headers) is
//     appended to DLQTopic and the offset is committed, so one poisoned event
//     never blocks the group.
type Consumer struct {
	reader  *kafka.Reader
	dlqW    *kafka.Writer // writer to DLQTopic
	handler Handler
	cfg     Config
}

var _ = (*Consumer)(nil)

// NewConsumer creates a member of the consumer group groupID for the platform
// topic. Multiple processes with the same groupID share partitions (scaling);
// distinct groupIDs (e.g. "audit", "webhooks") each get their own copy.
func NewConsumer(cfg Config, groupID string, handler Handler) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("events: no kafka brokers configured")
	}
	if groupID == "" {
		return nil, fmt.Errorf("events: consumer group id required")
	}
	if handler == nil {
		return nil, fmt.Errorf("events: handler required")
	}
	if cfg.Topic == "" {
		cfg.Topic = DefaultTopic
	}
	if cfg.DLQTopic == "" {
		cfg.DLQTopic = DefaultDLQTopic
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        groupID,
		GroupTopics:    []string{cfg.Topic},
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		// Bound how long an idle fetch waits (default 10s) so a low-volume
		// backbone delivers events promptly instead of in slow polls.
		MaxWait: 500 * time.Millisecond,
		// Where a NEW group starts (ignored once the group has committed
		// offsets). Auditors default to FirstOffset (full backlog) but the
		// audit-service overrides via Config to LastOffset so it does not
		// replay history on first deploy.
		StartOffset: cfg.ConsumerStartOffset,
	})
	dlqW := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Topic:                  cfg.DLQTopic,
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           50 * time.Millisecond,
		AllowAutoTopicCreation: true,
	}
	return &Consumer{reader: reader, dlqW: dlqW, handler: handler, cfg: cfg}, nil
}

// Run consumes until ctx is cancelled, then returns nil. A fatal Kafka error
// is returned (network partition etc) and the caller decides whether to exit.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, kafka.ErrGroupClosed) {
				return nil
			}
			return fmt.Errorf("events: fetch: %w", err)
		}

		ev, perr := c.decode(msg)
		if perr != nil {
			// Undecodable envelope -> straight to DLQ, don't block the group.
			log.Printf("events: undecodable message -> DLQ: %v", perr)
			c.dlq(ctx, msg)
			c.commitOrLog(ctx, msg)
			continue
		}

		herr := c.dispatch(ctx, ev)
		if herr != nil {
			log.Printf("events: handler failed for %s after retries: %v -> DLQ", ev.Type, herr)
			c.dlq(ctx, msg)
		}
		c.commitOrLog(ctx, msg)
	}
}

// dispatch invokes handler with retry+backoff.
func (c *Consumer) dispatch(ctx context.Context, ev CloudEvent) error {
	var lastErr error
	attempts := c.cfg.ConsumerMaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	for i := 0; i < attempts; i++ {
		lastErr = c.handler.Handle(ctx, ev)
		if lastErr == nil {
			return nil
		}
		if i < attempts-1 {
			select {
			case <-time.After(c.cfg.RetryBackoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return lastErr
}

// decode unmarshals a CloudEvent from the Kafka message value.
func (c *Consumer) decode(msg kafka.Message) (CloudEvent, error) {
	return UnmarshalCloudEvent(msg.Value)
}

// dlq appends the raw message (headers preserved) to the DLQ topic.
func (c *Consumer) dlq(ctx context.Context, msg kafka.Message) {
	dlqMsg := kafka.Message{Key: msg.Key, Value: msg.Value, Headers: msg.Headers}
	if err := c.dlqW.WriteMessages(ctx, dlqMsg); err != nil {
		log.Printf("events: DLQ write failed: %v", err)
	}
}

// commitOrLog commits the message offset, logging (not panicking) on failure —
// a failed commit means at-least-once redelivery, which the handler must handle.
func (c *Consumer) commitOrLog(ctx context.Context, msg kafka.Message) {
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		log.Printf("events: commit failed (will redeliver): %v", err)
	}
}

// Close releases the reader and DLQ writer.
func (c *Consumer) Close() error {
	var first error
	if c.reader != nil {
		if err := c.reader.Close(); err != nil {
			first = err
		}
	}
	if c.dlqW != nil {
		if err := c.dlqW.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
