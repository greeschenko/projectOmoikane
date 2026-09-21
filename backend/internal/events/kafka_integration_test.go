package events

// Kafka integration tests — Phase 28 gate: "event round-trips Kafka in
// compose; relay publishes, consumer acks".
//
// These run as part of `make go-test` (host-local, broker at localhost:9092
// via compose) and skip cleanly when Kafka is not reachable so CI/dev without
// a broker stays green. When the broker IS up they are real end-to-end tests:
// producer -> consumer group ack, outbox relay -> consumer, and DLQ routing.

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// brokerDefault is what the compose file publishes to the host: docker maps
// Kafka's 9092 (PLAINTEXT, advertised as localhost:9092) to localhost:9092.
const brokerDefault = "localhost:9092"

// kafkaIntegrationConfig returns a Config aimed at the host broker. If
// KAFKA_BROKERS is set (CI/compose), honor it; otherwise default to localhost.
func kafkaIntegrationConfig(t *testing.T) Config {
	t.Helper()
	cfg := ConfigFromEnv()
	if len(cfg.Brokers) == 0 || (len(cfg.Brokers) == 1 && cfg.Brokers[0] == "kafka:9092") {
		cfg.Brokers = []string{brokerDefault}
	}
	// Short retry/backoff so failure paths finish fast in tests.
	cfg.ConsumerMaxRetries = 1
	cfg.RetryBackoff = 50 * time.Millisecond
	cfg.OutboxMaxAttempts = 3
	return cfg
}

// kafkaReachable reports whether the broker answers a TCP dial within 3s.
func kafkaReachable(cfg Config) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := kafka.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// uniqueTopic returns a per-test topic so concurrent test runs never share
// offsets or leftover messages.
func uniqueTopic(prefix string) string {
	return fmt.Sprintf("%s.%s.%d", prefix, strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", ""), time.Now().UnixNano()%100000)
}

// provisionTopics is what a real service does at startup (before starting its
// producer/consumer): it ensures the platform topic + its DLQ topic exist.
func provisionTopics(t *testing.T, ctx context.Context, cfg Config) {
	t.Helper()
	if err := EnsureTopics(ctx, cfg.Brokers, []string{cfg.Topic, cfg.DLQTopic}); err != nil {
		t.Fatalf("provision topics: %v", err)
	}
}

// waitEvent waits up to timeout for a CloudEvent on ch (mirrors how a real
// consumer-group member gets an event and acks it).
func waitEvent(t *testing.T, ch chan CloudEvent, timeout time.Duration) CloudEvent {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for event (%v)", timeout)
		return CloudEvent{}
	}
}

// TestKafkaRoundTrip_PublishConsumeAck is the Phase 28 gate's core: a produced
// CloudEvent round-trips through Kafka and a consumer-group member acks it.
func TestKafkaRoundTrip_PublishConsumeAck(t *testing.T) {
	cfg := kafkaIntegrationConfig(t)
	if !kafkaReachable(cfg) {
		t.Skipf("kafka not reachable at %v — skipping integration test", cfg.Brokers)
	}

	topic := uniqueTopic("omoikane.it.roundtrip")
	cfg.Topic = topic
	cfg.DLQTopic = topic + ".dlq"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provisionTopics(t, ctx, cfg)

	producer, err := NewProducer(cfg)
	if err != nil {
		t.Fatalf("producer: %v", err)
	}
	defer producer.Close()

	type result struct {
		events chan CloudEvent
		errs   chan error
	}
	got := result{
		events: make(chan CloudEvent, 1),
		errs:   make(chan error, 1),
	}

	consumer, err := NewConsumer(cfg, "omoikane-it-roundtrip-"+topic,
		HandlerFunc(func(ctx context.Context, ev CloudEvent) error {
			got.events <- ev
			return nil
		}))
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	go func() {
		if err := consumer.Run(ctx); err != nil {
			got.errs <- err
		}
	}()

	// Publish first (topic auto-created), then let the consumer group start.
	ev := CloudEvent{
		SpecVersion: "1.0",
		ID:          "it-roundtrip-1",
		Source:      SourceContent,
		Type:        TypePostPublished,
		Subject:     "post/7",
		Time:        time.Now().UTC(),
		Data:        []byte(`{"id":7,"status":"published"}`),
	}
	if err := producer.Publish(ctx, ev); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case rcv := <-got.events:
		if rcv.ID != ev.ID || rcv.Type != ev.Type || rcv.Subject != ev.Subject {
			t.Fatalf("round-trip mismatch: got %+v want %+v", rcv, ev)
		}
	case err := <-got.errs:
		t.Fatalf("consumer run error: %v", err)
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for consumer ack")
	}

	// The consumer group committed; a second fetch should consume nothing more
	// (offset advanced). Give commit interval slack.
	time.Sleep(1500 * time.Millisecond)
	select {
	case extra := <-got.events:
		t.Fatalf("unexpected duplicate delivery: %+v", extra)
	default:
	}
}

// TestKafkaRoundTrip_OutboxRelayPublishes proves the outbox->relay->Kafka path:
// rows enqueued into the outbox are published by the relay and consumed by a
// consumer-group member (rack-accept flow end to end, DB through consumer).
func TestKafkaRoundTrip_OutboxRelayPublishes(t *testing.T) {
	cfg := kafkaIntegrationConfig(t)
	if !kafkaReachable(cfg) {
		t.Skipf("kafka not reachable at %v — skipping integration test", cfg.Brokers)
	}

	db := setupOutboxDB(t)
	store := NewGormOutboxStore(db)

	topic := uniqueTopic("omoikane.it.relay")
	cfg.Topic = topic
	cfg.DLQTopic = topic + ".dlq"
	cfg.RelayBatchSize = 10

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provisionTopics(t, ctx, cfg)

	producer, err := NewProducer(cfg)
	if err != nil {
		t.Fatalf("producer: %v", err)
	}
	defer producer.Close()

	// Enqueue 3 events directly via the outbox store (no business tx in test).
	want := []CloudEvent{
		{SpecVersion: "1.0", ID: "relay-a", Source: SourceAuth, Type: TypeUserRegistered, Subject: "user/1", Time: time.Now().UTC(), Data: []byte(`{"id":1}`)},
		{SpecVersion: "1.0", ID: "relay-b", Source: SourceContent, Type: TypePagePublished, Subject: "page/2", Time: time.Now().UTC(), Data: []byte(`{"id":2}`)},
		{SpecVersion: "1.0", ID: "relay-c", Source: SourceMedia, Type: TypeMediaUploaded, Subject: "media/3", Time: time.Now().UTC(), Data: []byte(`{"id":3}`)},
	}
	for _, ev := range want {
		if _, err := store.Enqueue(ctx, ev); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	relay := NewRelay(store, producer, cfg)
	if n, err := relay.RunOnce(ctx); err != nil {
		t.Fatalf("relay run: %v", err)
	} else if n != len(want) {
		t.Fatalf("relay attempted %d, want %d", n, len(want))
	}

	// All outbox rows must now be sent.
	var sent, failed int64
	var rows []OutboxEvent
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("load rows: %v", err)
	}
	for _, r := range rows {
		switch r.Status {
		case OutboxSent:
			sent++
		case OutboxFailed:
			failed++
		}
	}
	if sent != int64(len(want)) || failed != 0 {
		t.Fatalf("after relay: sent=%d failed=%d want sent=%d", sent, failed, len(want))
	}

	// Consume the 3 events; assert they arrive in publish order.
	received := make(chan CloudEvent, int64(len(want)))
	consumer, err := NewConsumer(cfg, "omoikane-it-relay-"+topic,
		HandlerFunc(func(ctx context.Context, ev CloudEvent) error {
			received <- ev
			return nil
		}))
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()
	go func() { _ = consumer.Run(ctx) }()

	seen := map[string]bool{}
	for i := 0; i < len(want); i++ {
		ev := waitEvent(t, received, 15*time.Second)
		if seen[ev.ID] {
			t.Fatalf("duplicate delivery: %s", ev.ID)
		}
		seen[ev.ID] = true
	}
	for _, w := range want {
		if !seen[w.ID] {
			t.Fatalf("did not receive %s", w.ID)
		}
	}
}

// TestKafkaRoundTrip_DLQ routes a constantly-failing handler's event to the
// DLQ topic after retries are exhausted, and the group keeps moving (the
// poisoned event does not wedge the consumer).
func TestKafkaRoundTrip_DLQ(t *testing.T) {
	cfg := kafkaIntegrationConfig(t)
	if !kafkaReachable(cfg) {
		t.Skipf("kafka not reachable at %v — skipping integration test", cfg.Brokers)
	}

	topic := uniqueTopic("omoikane.it.dlq")
	cfg.Topic = topic
	cfg.DLQTopic = topic + ".dlq"
	cfg.ConsumerMaxRetries = 1 // attempt + 1 retry, then DLQ

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provisionTopics(t, ctx, cfg)

	producer, err := NewProducer(cfg)
	if err != nil {
		t.Fatalf("producer: %v", err)
	}
	defer producer.Close()

	var handled atomic.Int32
	consumer, err := NewConsumer(cfg, "omoikane-it-dlq-"+topic,
		HandlerFunc(func(ctx context.Context, ev CloudEvent) error {
			handled.Add(1)
			return fmt.Errorf("permanent handler failure (simulated)")
		}))
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()
	go func() { _ = consumer.Run(ctx) }()

	ev := CloudEvent{
		SpecVersion: "1.0",
		ID:          "it-dlq-1",
		Source:      SourceMessages,
		Type:        TypeContactReceived,
		Subject:     "contact/9",
		Time:        time.Now().UTC(),
		Data:        []byte(`{"id":9}`),
	}
	if err := producer.Publish(ctx, ev); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Read the DLQ topic directly (no group) until the event shows up.
	dlqReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.DLQTopic,
		// No GroupID => consume from the start of the shared DLQ partition.
		StartOffset: kafka.FirstOffset,
		MaxWait:     500 * time.Millisecond,
	})
	defer dlqReader.Close()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		msg, err := dlqReader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				t.Fatalf("dlq fetch: %v", err)
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}
		rcv, err := UnmarshalCloudEvent(msg.Value)
		if err != nil {
			t.Fatalf("dlq unmarshal: %v", err)
		}
		if rcv.ID == ev.ID && rcv.Type == ev.Type {
			// Assert the type header survived into the DLQ copy.
			var hasType bool
			for _, h := range msg.Headers {
				if h.Key == "ce-type" && string(h.Value) == ev.Type {
					hasType = true
				}
			}
			if !hasType {
				t.Fatal("dlq message lost ce-type header")
			}
			if handled.Load() < 2 {
				t.Fatalf("expected handler to be retried (>=2 invocations), got %d", handled.Load())
			}
			return
		}
	}
	t.Fatal("event did not reach the DLQ topic in time")
}
