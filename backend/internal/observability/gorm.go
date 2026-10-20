package observability

import (
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// DefaultGormSlowThreshold is the boundary above which a statement is reported
// as slow (warn) instead of silently succeeding.
const DefaultGormSlowThreshold = 200 * time.Millisecond

// GormLogger returns a GORM logger that writes into the SAME structured JSON
// pipeline as every other service log line.
//
// GORM's default logger owns a separate stdlib logger on stderr and formats
// its own multi-line, ANSI-coloured text, so it escapes Setup()'s stdlib-log
// bridge completely: DB errors — and ordinary "record not found" control
// flow, which GORM reports at error level — showed up as raw unstructured
// lines next to otherwise-JSON logs (Phase 37 measured 62 of the audit
// service's 66 lines that way, and 26 of content's 190). Feeding stdlib `log`
// instead would not help either: its format embeds newlines, which the bridge
// splits into several half-records.
//
// GORM v1.31 ships a slog-backed logger (logger.NewSlogLogger), so we wire it
// to the service's default slog logger — the one Setup() installed with the
// `service` attribute — instead of hand-rolling a bridge:
//
//   - failed statement              -> error record (SQL + error attached)
//   - slow statement (> threshold)  -> warn record
//   - "record not found"            -> dropped: normal control flow, not an
//     error — logging in with an unknown email, reading settings before the
//     first write, a webhook delivery lookup that finds nothing
//   - routine statement             -> info, only at the Info level
//
// Pass logger.Warn (what every service uses) to keep routine SQL out of the
// logs while retaining failures and slow statements.
func GormLogger(level logger.LogLevel) logger.Interface {
	return logger.NewSlogLogger(slog.Default(), logger.Config{
		SlowThreshold:             DefaultGormSlowThreshold,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true, // expected control flow, never an error
	})
}
