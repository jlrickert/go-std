package std

import (
	"context"
	"maps"
	"os"
	"sort"
	"strings"
)

// Env is a small, descriptive interface for reading and modifying
// environment values. Implementations may read from the real process
// environment (OsEnv) or from an in-memory store useful for tests
// (TestEnv).
//
// The interface intentionally mirrors common environment operations so
// callers can easily swap a test implementation in unit tests without
// touching the real process environment.
type Env interface {
	// Get returns the raw environment value for key. The return value may
	// be empty when the key is not present.
	Get(key string) string

	// Set assigns the environment key to value.
	Set(key, value string) error

	// Has reports whether the environment key is set.
	Has(key string) bool

	// Environ returns a copy of the environment as a slice of strings in
	// the form "KEY=VALUE".
	Environ() []string

	// Unset removes the environment key.
	Unset(key string)

	// GetHome returns the user's home directory. Implementations should
	// return an error if the value is not available.
	GetHome() (string, error)

	// SetHome sets the user's home directory in the environment.
	SetHome(home string) error

	// GetUser returns the current user's username. Implementations should
	// return an error if the value is not available.
	GetUser() (string, error)

	// SetUser sets the current user's username in the environment.
	SetUser(user string) error

	// Getwd returns the working directory as seen by this Env. For OsEnv
	// this is the process working directory; for TestEnv it is the stored
	// PWD.
	Getwd() (string, error)

	// Setwd sets the working directory for this Env. For OsEnv this may
	// change the process working directory; for TestEnv it updates the
	// stored PWD.
	Setwd(dir string)

	// GetTempDir returns an appropriate temp directory for this Env. For
	// OsEnv this delegates to os.TempDir(); TestEnv provides testable
	// fallbacks.
	GetTempDir() string

	// ReadFile reads the named file from the filesystem view held by this
	// Env.
	ReadFile(name string) ([]byte, error)

	// WriteFile writes data to a file in the filesystem view held by this
	// Env using the provided permissions.
	WriteFile(name string, data []byte, perm os.FileMode) error

	// Mkdir creates a directory. If all is true MkdirAll is used.
	Mkdir(path string, perm os.FileMode, all bool) error

	// Remove removes the file or directory. If all is true RemoveAll is used.
	Remove(path string, all bool) error

	// Rename renames (moves) a file or directory.
	Rename(src, dst string) error
}

// GetDefault returns the value of key from the provided Env if present
// (non-empty), otherwise it returns the provided fallback value. Use this
// helper to prefer an environment value while allowing a default.
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
// environment visible via the Env stored in ctx. Each line is formatted
// as "KEY=VALUE".
//
// For TestEnv and OsEnv the function enumerates the known keys. For other
// Env implementations the function attempts to use common helper methods
// (Environ or Keys) if available; otherwise it returns a short message
// indicating enumeration is not possible.
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

// WithEnv returns a copy of ctx that carries the provided Env. Use this to
// inject a test environment into code under test.
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
