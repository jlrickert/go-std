package std

import (
	"context"
	"io"
	"maps"
	"os"
	"sort"
	"strings"
)

// Stream models the standard IO streams and common stream properties.
//
// Struct field tags are included for clarity to external consumers that may
// wish to encode some stream metadata. The actual reader and writer fields
// are not suitable for encoding and therefore are tagged to be ignored.
type Stream struct {
	In  io.Reader // input stream, for example os.Stdin
	Out io.Writer // output stream, for example os.Stdout
	Err io.Writer // error stream, for example os.Stderr

	IsPiped bool //  indicates whether stdin appears to be piped or redirected.
	IsTTY   bool // indicates whether stdout refers to a terminal.
}

// Env is a compact interface for reading and modifying environment
// values. Implementations may reflect the real process environment (OsEnv)
// or provide an in-memory view suitable for tests (TestEnv).
//
// The interface mirrors common environment operations so callers can inject a
// test implementation for unit tests without touching the real process env.
type Env interface {
	// Get returns the raw environment value for key. The return value may be
	// empty when the key is not present.
	Get(key string) string

	// Set assigns the environment key to value.
	Set(key, value string) error

	// Has reports whether the environment key is set.
	Has(key string) bool

	// Environ returns a copy of the environment as a slice of strings in the
	// form "KEY=VALUE".
	Environ() []string

	// Unset removes the environment key.
	Unset(key string)

	// GetHome returns the user's home directory. Implementations should return
	// an error if the value is not available.
	GetHome() (string, error)

	// SetHome sets the user's home directory in the environment.
	SetHome(home string) error

	// GetUser returns the current user's username. Implementations should
	// return an error if the value is not available.
	GetUser() (string, error)

	// SetUser sets the current user's username in the environment.
	SetUser(user string) error

	// Getwd returns the working directory as seen by this Env. For OsEnv this
	// is the process working directory; for TestEnv it is the stored PWD.
	Getwd() (string, error)

	// Setwd sets the working directory for this Env. For OsEnv this may change
	// the process working directory; for TestEnv it updates the stored PWD.
	Setwd(dir string)

	// GetTempDir returns an appropriate temp directory for this Env. For OsEnv
	// this delegates to os.TempDir(); TestEnv provides testable fallbacks.
	GetTempDir() string

	// Stdio returns the input reader for this Env.
	Stdio() io.Reader
	// Stdout returns the output writer for this Env.
	Stdout() io.Writer
	// Stderr returns the error writer for this Env.
	Stderr() io.Writer

	// IsTTY reports whether stdout is a terminal.
	IsTTY() bool
	// IsStdioPiped reports whether stdin appears piped or redirected.
	IsStdioPiped() bool
}

// GetDefault returns the value of key from env when present and non-empty.
// Otherwise it returns the provided fallback value. Use this helper when a
// preference for an environment value is desired while still allowing a
// concrete default.
func GetDefault(env Env, key, other string) string {
	if env == nil {
		return other
	}
	if v := env.Get(key); v != "" {
		return v
	}
	return other
}

// ExpandEnv expands $var or ${var} in s using the Env stored in ctx. If no
// Env is present in the context the real OS environment is used via OsEnv.
func ExpandEnv(ctx context.Context, s string) string {
	env := EnvFromContext(ctx)
	return os.Expand(s, env.Get)
}

// DumpEnv returns a sorted, newline separated representation of the
// environment visible via the Env stored in ctx. Each line is formatted as
// "KEY=VALUE".
//
// For TestEnv and OsEnv the function enumerates the known keys. For other Env
// implementations the function attempts to use common helper methods (Environ
// or Keys) if available. If enumeration is not possible a short message is
// returned indicating that limitation.
func DumpEnv(ctx context.Context) string {
	env := EnvFromContext(ctx)
	entries := make(map[string]string)

	// Special-case TestEnv to expose its map and dedicated HOME/USER fields.
	if te, ok := env.(*TestEnv); ok {
		if te.data != nil {
			maps.Copy(entries, te.data)
		}
		if te.home != "" {
			entries["HOME"] = te.home
		}
		if te.user != "" {
			entries["USER"] = te.user
		}
	} else if _, ok := env.(*OsEnv); ok {
		// OsEnv: fall back to the process environment.
		for _, kv := range os.Environ() {
			if i := strings.Index(kv, "="); i >= 0 {
				entries[kv[:i]] = kv[i+1:]
			}
		}
	} else if en, ok := env.(interface{ Environ() []string }); ok {
		// Generic Environ() method returning "KEY=VAL" strings.
		for _, kv := range en.Environ() {
			if i := strings.Index(kv, "="); i >= 0 {
				entries[kv[:i]] = kv[i+1:]
			}
		}
	} else if ks, ok := env.(interface{ Keys() []string }); ok {
		// Generic Keys() method returning a list of keys.
		for _, k := range ks.Keys() {
			entries[k] = env.Get(k)
		}
	} else {
		return "env: cannot enumerate keys for this Env implementation"
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(entries[k])
		b.WriteByte('\n')
	}
	return b.String()
}

type envCtxKey int

var (
	ctxEnvKey  envCtxKey
	defaultEnv = &OsEnv{}
)

// WithEnv returns a copy of ctx that carries env. Use this to inject a test
// environment into code under test.
func WithEnv(ctx context.Context, env Env) context.Context {
	return context.WithValue(ctx, ctxEnvKey, env)
}

// EnvFromContext returns the Env stored in ctx. If ctx is nil or does not
// contain an Env, the real OsEnv is returned.
func EnvFromContext(ctx context.Context) Env {
	if v := ctx.Value(ctxEnvKey); v != nil {
		if env, ok := v.(Env); ok && env != nil {
			return env
		}
	}
	return defaultEnv
}

// StreamFromContext builds a Stream summarizing stdio and related properties
// from the Env stored in ctx.
func StreamFromContext(ctx context.Context) *Stream {
	env := EnvFromContext(ctx)

	return &Stream{
		In:      env.Stdio(),
		Out:     env.Stdout(),
		Err:     env.Stderr(),
		IsPiped: env.IsStdioPiped(),
		IsTTY:   env.IsTTY(),
	}
}
