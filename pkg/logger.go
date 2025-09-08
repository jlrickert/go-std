package std

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"log/slog"
)

// Package std provides small logging utilities built on slog for tests and
// simple applications. It includes helpers to create configured loggers,
// utilities to store/retrieve loggers from contexts, and a test handler that
// captures structured log entries for assertions in tests.

// ParseLevel maps common textual level names to slog.Level. The function is
// case-insensitive and ignores surrounding whitespace. If an unrecognized value
// is provided, slog.LevelInfo is returned.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LoggerConfig is a minimal, convenient set of options for creating a new
// slog.Logger.
//
// Fields:
//   - Version: application or build version included with each log entry.
//   - Out: destination writer for log output. If nil, os.Stdout is used.
//   - Level: minimum logging level.
//   - JSON: when true, output is JSON; otherwise, human-readable text is used.
type LoggerConfig struct {
	Version string

	// If Out is nil, stdout is used.
	Out io.Writer

	Level slog.Level
	JSON  bool // true => JSON output, false => text
}

// NewLogger creates a configured *slog.Logger and a shutdown function.
// The shutdown function is a no-op in this implementation but is returned to
// make it easy to add asynchronous or file-based writers later. Call the
// shutdown function on process exit if you add asynchronous writers.
func NewLogger(cfg LoggerConfig) *slog.Logger {
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}

	var handler slog.Handler
	if cfg.JSON {
		handler = slog.NewJSONHandler(
			out,
			&slog.HandlerOptions{Level: cfg.Level, AddSource: true})
	} else {
		handler = slog.NewTextHandler(
			out,
			&slog.HandlerOptions{Level: cfg.Level, AddSource: true})
	}

	attrs := []slog.Attr{
		slog.String("version", cfg.Version),
	}
	hn, _ := os.Hostname()
	attrs = append(attrs, slog.Int("pid", os.Getpid()))

	logger := slog.New(handler).With(
		slog.String("version", cfg.Version),
		slog.String("host", hn),
	)

	// shutdown noop for now
	return logger
}

// NewDiscardLogger returns a logger whose output is discarded. This is useful for
// tests where log output should be suppressed.
func NewDiscardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

///////////////////////////////////////////////////////////////////////////////
// Context helpers
///////////////////////////////////////////////////////////////////////////////

// ctxKeyType is an unexported type used as the context key to avoid collisions.
type ctxKeyType struct{}

var ctxKey ctxKeyType

// ContextWithLogger returns a copy of ctx that carries the provided logger.
// Use this to associate a logger with a context for downstream callers.
func ContextWithLogger(ctx context.Context, lg *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey, lg)
}

// FromContext returns the logger stored in ctx. If ctx is nil or does not
// contain a logger, slog.Default() is returned.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if v := ctx.Value(ctxKey); v != nil {
		if lg, ok := v.(*slog.Logger); ok && lg != nil {
			return lg
		}
	}
	return slog.Default()
}

// GetLogger returns a logger from the context if present. If no logger is
// stored on the context, the provided fallback is returned. If fallback is
// nil, slog.Default() is returned.
func GetLogger(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if lg := FromContext(ctx); lg != nil {
		return lg
	}
	if fallback != nil {
		return fallback
	}
	return slog.Default()
}

///////////////////////////////////////////////////////////////////////////////
// Test handler (simple, thread-safe)
///////////////////////////////////////////////////////////////////////////////

// LoggedEntry represents a captured structured log entry for assertions in
// tests. It contains the timestamp, level, message and any attributes.
type LoggedEntry struct {
	Time  time.Time
	Level slog.Level
	Msg   string
	Attrs map[string]any
}

// testingT is a tiny subset of *testing.T used for optional logging from the
// test handler. Only Logf is required.
type testingT interface {
	Logf(format string, args ...any)
}

// TestHandler captures structured entries so tests can assert on logs. It is
// safe for concurrent use.
type TestHandler struct {
	mu      sync.Mutex
	Entries []LoggedEntry
	T       testingT
}

// NewTestHandler creates an empty TestHandler. Optionally pass a testing.T to
// have the handler echo captured entries to the test log (via Logf).
func NewTestHandler(t testingT) *TestHandler {
	return &TestHandler{T: t}
}

// Enabled returns true for all levels. Filtering is expected to be handled by
// the caller or the logger's handler options.
func (h *TestHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

// Handle captures the provided record as a LoggedEntry and appends it to the
// handler's Entries slice. If a testingT was provided, a human-readable line
// is also logged to the test output.
func (h *TestHandler) Handle(ctx context.Context, r slog.Record) error {
	e := LoggedEntry{
		Time:  r.Time,
		Level: r.Level,
		Msg:   r.Message,
		Attrs: map[string]any{},
	}
	h.mu.Lock()
	h.Entries = append(h.Entries, e)
	h.mu.Unlock()

	if h.T != nil {
		h.T.Logf("LOG %s %v %v", e.Msg, e.Level, e.Attrs)
	}
	return nil
}

// WithAttrs returns the handler unchanged. Attributes are captured per record
// in Handle, so no additional state is needed here.
func (h *TestHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }

// WithGroup returns the handler unchanged. Grouping is not modeled by this
// simple test handler.
func (h *TestHandler) WithGroup(_ string) slog.Handler { return h }

// NewTestLogger returns a *slog.Logger that writes to a TestHandler and the
// handler itself for assertions. The returned logger has a default attribute
// ("test"="true") to make it easier to identify test logs.
func NewTestLogger(t testingT, level slog.Level) (*slog.Logger, *TestHandler) {
	th := NewTestHandler(t)
	logger := slog.New(th).With(slog.String("test", "true"))
	return logger, th
}

var _ slog.Handler = (*TestHandler)(nil)

///////////////////////////////////////////////////////////////////////////////
// Small helpers for tests
///////////////////////////////////////////////////////////////////////////////

// FindEntries returns a copy of entries from the TestHandler that satisfy the
// provided predicate. The handler's internal slice is copied under lock to
// avoid races.
func FindEntries(th *TestHandler, pred func(LoggedEntry) bool) []LoggedEntry {
	th.mu.Lock()
	entries := append([]LoggedEntry(nil), th.Entries...)
	th.mu.Unlock()

	out := make([]LoggedEntry, 0)
	for _, e := range entries {
		if pred(e) {
			out = append(out, e)
		}
	}
	return out
}

// RequireEntry fails the test if a matching entry isn't observed within the
// given timeout. When a matching entry is found it is returned. If the timeout
// elapses, the test is failed and the captured entries are included in the
// failure message to aid debugging.
func RequireEntry(t *testing.T, th *TestHandler, pred func(LoggedEntry) bool, timeout time.Duration) LoggedEntry {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		th.mu.Lock()
		for _, e := range th.Entries {
			if pred(e) {
				out := e
				th.mu.Unlock()
				return out
			}
		}
		th.mu.Unlock()
		if time.Now().After(deadline) {
			th.mu.Lock()
			entries := append([]LoggedEntry(nil), th.Entries...)
			th.mu.Unlock()
			t.Fatalf("required log entry not found in %s; captured %d entries: %#v", timeout, len(entries), entries)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
