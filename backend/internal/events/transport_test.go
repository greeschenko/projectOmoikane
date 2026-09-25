package events

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/segmentio/kafka-go/sasl/plain"
)

// Managed-broker security tests (Phase 36): KAFKA_SECURITY_PROTOCOL /
// KAFKA_SASL_* parsing + dialer/transport construction. None of these touch a
// broker — they assert configuration only.

func TestConfigFromEnv_KafkaSecurity(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "broker.example:9096")
	t.Setenv("KAFKA_SECURITY_PROTOCOL", "SASL_SSL")
	t.Setenv("KAFKA_SASL_MECHANISM", "PLAIN")
	t.Setenv("KAFKA_SASL_USERNAME", "user1")
	t.Setenv("KAFKA_SASL_PASSWORD", "secret1")

	cfg := ConfigFromEnv()
	if cfg.SecurityProtocol != "SASL_SSL" || cfg.SASLMechanism != "PLAIN" ||
		cfg.SASLUsername != "user1" || cfg.SASLPassword != "secret1" {
		t.Fatalf("ConfigFromEnv did not pick up KAFKA_* security envs: %+v", cfg)
	}
	if len(cfg.Brokers) != 1 || cfg.Brokers[0] != "broker.example:9096" {
		t.Fatalf("brokers wrong: %v", cfg.Brokers)
	}
}

func TestConfigFromEnv_KafkaSecurityDefaults(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	// Ensure no stray env leaks from the host into the default case.
	t.Setenv("KAFKA_SECURITY_PROTOCOL", "")
	t.Setenv("KAFKA_SASL_MECHANISM", "")
	cfg := ConfigFromEnv()
	if cfg.SecurityProtocol != "" || cfg.SASLMechanism != "" || cfg.SASLUsername != "" {
		t.Fatalf("default Config must be PLAINTEXT, got %+v", cfg)
	}
}

func TestValidateSecurity(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"plaintext default", Config{}, false},
		{"SSL alone", Config{SecurityProtocol: "SSL"}, false},
		{"SASL_SSL missing mechanism", Config{SecurityProtocol: "SASL_SSL"}, true},
		{"SASL_SSL + mechanism, no creds",
			Config{SecurityProtocol: "SASL_SSL", SASLMechanism: "PLAIN"}, true},
		{"SASL_SSL + PLAIN + creds",
			Config{SecurityProtocol: "SASL_SSL", SASLMechanism: "PLAIN", SASLUsername: "u", SASLPassword: "p"}, false},
		{"mechanism but plaintext protocol",
			Config{SASLMechanism: "PLAIN", SASLUsername: "u", SASLPassword: "p"}, true},
		{"unknown protocol", Config{SecurityProtocol: "GSSAPI"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validateSecurity()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSecurity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSASLMechanism(t *testing.T) {
	cfg := Config{SASLMechanism: "PLAIN", SASLUsername: "u", SASLPassword: "p"}
	m, err := cfg.saslMechanism()
	if err != nil {
		t.Fatalf("PLAIN mechanism: %v", err)
	}
	p, ok := m.(plain.Mechanism)
	if !ok || p.Username != "u" || p.Password != "p" {
		t.Fatalf("PLAIN mechanism wrong: %#v", m)
	}

	for _, mech := range []string{"SCRAM-SHA-256", "SCRAM-SHA-512"} {
		m, err := (Config{SASLMechanism: mech, SASLUsername: "u", SASLPassword: "p"}).saslMechanism()
		if err != nil {
			t.Fatalf("%s mechanism: %v", mech, err)
		}
		if m == nil || m.Name() != mech {
			t.Fatalf("%s mechanism name wrong: %v", mech, m)
		}
	}

	if _, err := (Config{SASLMechanism: "AWS_MSK_IAM"}).saslMechanism(); err == nil {
		t.Fatal("unsupported mechanism must error (AWS_MSK_IAM needs the signer package — out of scope)")
	}
}

func TestTLSConfig(t *testing.T) {
	if cfg := (Config{}).tlsConfig(); cfg != nil {
		t.Fatalf("PLAINTEXT must have nil TLS, got %v", cfg)
	}
	for _, proto := range []string{"SSL", "SASL_SSL"} {
		cfg := (Config{SecurityProtocol: proto}).tlsConfig()
		if cfg == nil {
			t.Fatalf("%s must produce TLS config", proto)
		}
		if cfg.MinVersion != tls.VersionTLS12 {
			t.Fatalf("%s MinVersion = %d, want TLS 1.2", proto, cfg.MinVersion)
		}
	}
}

func TestNewProducer_RejectsUnsupportedSecurity(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SecurityProtocol = "SASL_SSL"
	cfg.SASLMechanism = "AWS_MSK_IAM"
	if _, err := NewProducer(cfg); err == nil {
		t.Fatal("NewProducer must reject unsupported SASL mechanism")
	}
}

func TestNewConsumer_DialerHonorsSecurity(t *testing.T) {
	// Construction only — no broker is contacted by NewReader on init.
	cfg := DefaultConfig()
	cfg.SecurityProtocol = "SASL_SSL"
	cfg.SASLMechanism = "PLAIN"
	cfg.SASLUsername = "u"
	cfg.SASLPassword = "p"
	c, err := NewConsumer(cfg, "test-group", HandlerFunc(func(ctx context.Context, ev CloudEvent) error { return nil }))
	if err != nil {
		t.Fatalf("NewConsumer with SASL_SSL config: %v", err)
	}
	defer c.Close()
	if c.reader == nil || c.dlqW == nil {
		t.Fatal("reader or DLQ writer nil")
	}

	// The same Config drives the builders; assert the security surface here
	// (we cannot reach into kafka.Reader's private dialer from outside kafka-go).
	d, err := cfg.newDialer()
	if err != nil {
		t.Fatalf("newDialer: %v", err)
	}
	if d.TLS == nil || d.SASLMechanism == nil {
		t.Fatal("dialer must carry TLS + SASL for SASL_SSL/PLAIN")
	}
	tr, err := cfg.newTransport()
	if err != nil {
		t.Fatalf("newTransport: %v", err)
	}
	if tr.TLS == nil || tr.SASL == nil {
		t.Fatal("transport must carry TLS + SASL for SASL_SSL/PLAIN")
	}
}
