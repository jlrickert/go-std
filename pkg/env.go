package std

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// Env is a small, descriptive interface for reading and modifying
// environment values. Implementations may read from the real process
// environment (OsEnv) or from an in-memory store useful for tests
// (MapEnv). The interface intentionally mirrors common environment
// operations so callers can easily swap a test implementation in unit
// tests without touching the real process environment.
type Env interface {
	// Get returns the raw environment value for key (may be empty).
	Get(key string) string

	// Set assigns the environment key to value.
	Set(key, value string) error

	Has(key string) bool

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

	// GetWd returns the working directory as seen by this Env. For OsEnv this
	// is the process working directory; for MapEnv it is the stored PWD.
	Getwd() (string, error)

	// SetWd sets the working directory for this Env. For OsEnv this may
	// change the process working directory; for MapEnv it updates the stored PWD.
	Setwd(dir string)

	// GetTempDir returns an appropriate temp directory for this Env. For OsEnv
	// this delegates to os.TempDir(); MapEnv provides testable fallbacks.
	GetTempDir() string

	WriteFile(name string, data []byte, perm os.FileMode) error

	Mkdir(path string, perm os.FileMode, all bool) error
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

// EnsureInJail returns a path that resides inside jail when possible.
//
// If p is already inside jail the cleaned absolute form of p is returned.
// Otherwise a path under jail is returned by appending the base name of p.
func EnsureInJail(jail, p string) string {
	if jail == "" {
		return p
	}
	// Clean inputs.
	j := filepath.Clean(jail)
	if p == "" {
		return j
	}
	pp := filepath.Clean(p)

	// If pp is relative, make it absolute relative to jail and return it.
	if !filepath.IsAbs(pp) {
		return filepath.Join(j, pp)
	}

	// If pp is within j, return pp as-is.
	rel, err := filepath.Rel(j, pp)
	if err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
		return pp
	}
	// Otherwise, place a safe fallback under jail using the base name.
	base := filepath.Base(pp)
	return filepath.Join(j, base)
}

// EnsureInJailFor is a test-friendly helper that mirrors EnsureInJail but
// accepts paths written with forward slashes. It converts both jail and p
// using filepath.FromSlash before applying the EnsureInJail logic.
//
// Use this from tests when expected values are easier to express using
// posix-style literals.
func EnsureInJailFor(jail, p string) string {
	// Convert slash-separated test inputs to platform-specific form.
	j := filepath.FromSlash(jail)
	pp := filepath.FromSlash(p)

	// Reuse EnsureInJail logic on the converted inputs. This ensures tests
	// see the same behavior as production code while allowing easier literals.
	return EnsureInJail(j, pp)
}
