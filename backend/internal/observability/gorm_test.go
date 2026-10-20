package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

// captureSlog installs a JSON default logger writing into a buffer and
// restores the previous one on cleanup, so tests can assert on records the
// same way services emit them.
func captureSlog(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func records(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not JSON: %v (%q)", err, line)
		}
		out = append(out, rec)
	}
	return out
}

// traceAttr digs the trace group out of a record, asserting it is there.
func traceAttr(t *testing.T, rec map[string]any) map[string]any {
	t.Helper()
	group, ok := rec["trace"].(map[string]any)
	if !ok {
		t.Fatalf("record has no trace group: %v", rec)
	}
	return group
}

// TestGormLogger_ErrorsAreStructuredJSON is the Phase 37 regression: GORM's
// stock logger owns a separate stderr logger, so DB errors escaped the JSON
// pipeline as raw ANSI text (62 of the audit service's 66 lines). They must
// now be single JSON ERROR records carrying the sql/error attributes.
func TestGormLogger_ErrorsAreStructuredJSON(t *testing.T) {
	buf := captureSlog(t, slog.LevelDebug)
	// The logger must be built AFTER the default handler is installed, the
	// same order services use (Setup first, then gorm.Open).
	lg := GormLogger(gormlogger.Warn)

	lg.Trace(context.Background(), time.Now(),
		func() (string, int64) { return `INSERT INTO "audit_logs" VALUES ($1)`, 1 },
		errors.New(`pq: duplicate key value violates unique constraint "idx_audit_logs_event_id"`))

	recs := records(t, buf)
	if len(recs) != 1 {
		t.Fatalf("expected exactly 1 record, got %d: %q", len(recs), buf.String())
	}
	rec := recs[0]
	if rec["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", rec["level"])
	}
	trace := traceAttr(t, rec)
	if sql, _ := trace["sql"].(string); !strings.Contains(sql, `INSERT INTO "audit_logs"`) {
		t.Errorf("trace.sql missing: %v", trace)
	}
	if errAttr, _ := trace["error"].(string); !strings.Contains(errAttr, "duplicate key") {
		t.Errorf("trace.error missing: %v", trace)
	}
	if trace["rows"].(float64) != 1 {
		t.Errorf("trace.rows = %v, want 1", trace["rows"])
	}
	if _, ok := trace["duration"].(string); !ok {
		t.Errorf("trace.duration missing: %v", trace)
	}
}

// TestGormLogger_RecordNotFoundIsNotAnError — logging in with an unknown email
// or reading settings before the first write is normal control flow. The stock
// logger reported it at ERROR level; it must produce NOTHING at the levels
// services run at (Warn), so a wrong-password login can never be mistaken for
// a database failure by log alerting.
func TestGormLogger_RecordNotFoundIsNotAnError(t *testing.T) {
	buf := captureSlog(t, slog.LevelDebug)

	for _, level := range []gormlogger.LogLevel{gormlogger.Error, gormlogger.Warn, gormlogger.Info} {
		buf.Reset()
		lg := GormLogger(level)
		lg.Trace(context.Background(), time.Now(),
			func() (string, int64) { return `SELECT * FROM "users" WHERE email = $1`, 0 },
			gormlogger.ErrRecordNotFound)
		for _, rec := range records(t, buf) {
			if rec["level"] == "ERROR" {
				t.Errorf("record not found logged as ERROR at %s level: %v", levelName(level), rec)
			}
		}
	}

	// At the level services actually run at, it must be silent entirely.
	buf.Reset()
	GormLogger(gormlogger.Warn).Trace(context.Background(), time.Now(),
		func() (string, int64) { return `SELECT * FROM "users"`, 0 },
		gormlogger.ErrRecordNotFound)
	if buf.Len() != 0 {
		t.Errorf("record not found logged at Warn level: %q", buf.String())
	}
}

func levelName(l gormlogger.LogLevel) string {
	switch l {
	case gormlogger.Silent:
		return "Silent"
	case gormlogger.Error:
		return "Error"
	case gormlogger.Warn:
		return "Warn"
	case gormlogger.Info:
		return "Info"
	}
	return "?"
}

// TestGormLogger_LevelFiltering — a Warn-level logger must keep failures and
// slow statements, and drop routine SQL entirely.
func TestGormLogger_LevelFiltering(t *testing.T) {
	buf := captureSlog(t, slog.LevelDebug)
	// DefaultGormSlowThreshold is 200ms; a statement begun 5s ago is slow.
	lg := GormLogger(gormlogger.Warn)

	// Routine, fast statement: nothing.
	lg.Trace(context.Background(), time.Now(),
		func() (string, int64) { return `SELECT 1`, 1 }, nil)
	if buf.Len() != 0 {
		t.Errorf("routine SQL logged at Warn level: %q", buf.String())
	}

	// Slow statement: warn.
	lg.Trace(context.Background(), time.Now().Add(-5*time.Second),
		func() (string, int64) { return `SELECT * FROM pages`, 10 }, nil)
	recs := records(t, buf)
	if len(recs) != 1 || recs[0]["level"] != "WARN" {
		t.Fatalf("slow statement should be 1 WARN record, got %d: %v", len(recs), recs)
	}

	// Silent logger emits nothing at all.
	buf.Reset()
	silent := GormLogger(gormlogger.Silent)
	silent.Trace(context.Background(), time.Now(),
		func() (string, int64) { return `SELECT 1`, 1 }, errors.New("boom"))
	if buf.Len() != 0 {
		t.Errorf("Silent level logged: %q", buf.String())
	}
}

// TestGormLogger_InfoLevelRecordsSQL — the full-detail mode still works, with
// the SQL text and row count attached to every statement.
func TestGormLogger_InfoLevelRecordsSQL(t *testing.T) {
	buf := captureSlog(t, slog.LevelDebug)
	lg := GormLogger(gormlogger.Info)

	lg.Trace(context.Background(), time.Now(),
		func() (string, int64) { return `SELECT * FROM "pages"`, 3 }, nil)
	recs := records(t, buf)
	if len(recs) != 1 || recs[0]["level"] != "INFO" {
		t.Fatalf("expected 1 INFO record, got %d: %v", len(recs), recs)
	}
	trace := traceAttr(t, recs[0])
	if rows, _ := trace["rows"].(float64); rows != 3 {
		t.Errorf("trace.rows = %v, want 3", trace["rows"])
	}
}

// TestGormLogger_ParamsFilterKeepsSQL — bound parameters stay in the text (the
// stock logger's behaviour), so failures remain diagnosable. GORM calls it
// through an optional interface assertion on the concrete logger, so the test
// asserts the same way callbacks.go does.
func TestGormLogger_ParamsFilterKeepsSQL(t *testing.T) {
	filter, ok := GormLogger(gormlogger.Warn).(interface {
		ParamsFilter(context.Context, string, ...interface{}) (string, []interface{})
	})
	if !ok {
		t.Fatal("GormLogger must expose ParamsFilter (gorm calls it via optional interface assertion)")
	}
	sql, params := filter.ParamsFilter(context.Background(), "SELECT * FROM users WHERE id = ?", 7)
	if sql != "SELECT * FROM users WHERE id = ?" {
		t.Errorf("sql = %q", sql)
	}
	if len(params) != 1 || params[0] != 7 {
		t.Errorf("params = %v (want them preserved, not parameterised away)", params)
	}
}

// TestGormLogger_LogModeClones — GORM clones on level change; the original
// must be unaffected.
func TestGormLogger_LogModeClones(t *testing.T) {
	orig := GormLogger(gormlogger.Warn)
	quiet := orig.LogMode(gormlogger.Silent)
	if quiet == orig {
		t.Fatal("LogMode must return a clone, not mutate in place")
	}
	buf := captureSlog(t, slog.LevelDebug)
	orig.Trace(context.Background(), time.Now(),
		func() (string, int64) { return `SELECT 1`, 1 }, errors.New("boom"))
	if buf.Len() == 0 {
		t.Error("original logger lost its level")
	}
	buf.Reset()
	quiet.Trace(context.Background(), time.Now(),
		func() (string, int64) { return `SELECT 1`, 1 }, errors.New("boom"))
	if buf.Len() != 0 {
		t.Errorf("cloned Silent logger logged: %q", buf.String())
	}
}

// TestGormLogger_WarnErrorMethods — the direct Info/Warn/Error entry points are
// routed too (application code and GORM internals both use them), with the
// right level and the message carried verbatim.
func TestGormLogger_WarnErrorMethods(t *testing.T) {
	buf := captureSlog(t, slog.LevelDebug)
	lg := GormLogger(gormlogger.Warn)

	lg.Error(context.Background(), "migration failed")
	lg.Warn(context.Background(), "connection pool saturated")
	lg.Info(context.Background(), "should be filtered at Warn level")

	recs := records(t, buf)
	if len(recs) != 2 {
		t.Fatalf("expected 2 records (info filtered at Warn), got %d: %v", len(recs), recs)
	}
	if recs[0]["level"] != "ERROR" || recs[1]["level"] != "WARN" {
		t.Errorf("levels = %v, %v", recs[0]["level"], recs[1]["level"])
	}
	if msg, _ := recs[0]["msg"].(string); msg != "migration failed" {
		t.Errorf("error msg = %q", recs[0]["msg"])
	}
}
