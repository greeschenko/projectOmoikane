package events

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// Transport construction (Phase 36): managed brokers (MSK with TLS/SASL,
// Confluent Cloud SASL_SSL) need a TLS+SASL-capable dialer/transport wired into
// the producer writer, consumer reader, DLQ writer and topic-provisioning
// client. With the defaults (no SecurityProtocol) everything stays PLAINTEXT
// and behaves exactly as before — these helpers only matter when
// KAFKA_SECURITY_PROTOCOL / KAFKA_SASL_* are set.
//
// TLS: hardened MinVersion; the platform trusts managed CA chains (RDS/MSK/
// Confluent) via the system pool. Per-broker custom CAs are out of scope for
// the platform-demo chart (documented in docs/cloud/*.md).

// tlsConfig returns a hardened TLS client config when the configured security
// protocol requires it ("SSL" or "SASL_SSL"); nil for PLAINTEXT.
func (cfg Config) tlsConfig() *tls.Config {
	switch cfg.SecurityProtocol {
	case "SSL", "SASL_SSL":
		return &tls.Config{MinVersion: tls.VersionTLS12}
	default:
		return nil
	}
}

// saslMechanism resolves the configured SASL mechanism for managed brokers
// (Confluent Cloud PLAIN, MSK SCRAM). Returns nil when no mechanism is set.
func (cfg Config) saslMechanism() (sasl.Mechanism, error) {
	if cfg.SASLMechanism == "" {
		return nil, nil
	}
	switch cfg.SASLMechanism {
	case "PLAIN":
		return plain.Mechanism{Username: cfg.SASLUsername, Password: cfg.SASLPassword}, nil
	case "SCRAM-SHA-256":
		return scram.Mechanism(scram.SHA256, cfg.SASLUsername, cfg.SASLPassword)
	case "SCRAM-SHA-512":
		return scram.Mechanism(scram.SHA512, cfg.SASLUsername, cfg.SASLPassword)
	default:
		return nil, fmt.Errorf("events: unsupported SASL mechanism %q (supported: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)", cfg.SASLMechanism)
	}
}

// validateSecurity rejects unusable security configurations early (at service
// startup), instead of every connection failing with an opaque broker error.
// A mechanism without credentials (or SSL without a mechanism name but with
// credentials) is almost certainly a misconfiguration.
func (cfg Config) validateSecurity() error {
	switch cfg.SecurityProtocol {
	case "", "SSL", "SASL_SSL":
		// known
	default:
		return fmt.Errorf("events: unsupported KAFKA_SECURITY_PROTOCOL %q (supported: SSL, SASL_SSL)", cfg.SecurityProtocol)
	}
	if cfg.SASLMechanism != "" {
		if cfg.SecurityProtocol != "SASL_SSL" {
			return fmt.Errorf("events: KAFKA_SASL_MECHANISM set but KAFKA_SECURITY_PROTOCOL is %q (want SASL_SSL)", cfg.SecurityProtocol)
		}
		if cfg.SASLUsername == "" || cfg.SASLPassword == "" {
			return fmt.Errorf("events: SASL mechanism %q requires KAFKA_SASL_USERNAME and KAFKA_SASL_PASSWORD", cfg.SASLMechanism)
		}
	}
	if cfg.SecurityProtocol == "SASL_SSL" && cfg.SASLMechanism == "" {
		return fmt.Errorf("events: KAFKA_SECURITY_PROTOCOL=SASL_SSL requires KAFKA_SASL_MECHANISM")
	}
	return nil
}

// newDialer builds the *kafka.Dialer used by consumers; nil-mechanism keeps
// the existing PLAINTEXT behavior.
func (cfg Config) newDialer() (*kafka.Dialer, error) {
	m, err := cfg.saslMechanism()
	if err != nil {
		return nil, err
	}
	return &kafka.Dialer{
		Timeout:       10 * time.Second,
		TLS:           cfg.tlsConfig(),
		SASLMechanism: m,
	}, nil
}

// newTransport builds the *kafka.Transport used by producers and the DLQ
// writer (writers dial through Transport instead of their own default).
func (cfg Config) newTransport() (*kafka.Transport, error) {
	m, err := cfg.saslMechanism()
	if err != nil {
		return nil, err
	}
	return &kafka.Transport{
		DialTimeout: 10 * time.Second,
		TLS:         cfg.tlsConfig(),
		SASL:        m,
	}, nil
}
