package toolkit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	std "github.com/jlrickert/go-std/toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefault(t *testing.T) {
	jail := t.TempDir()
	env := std.NewTestEnv(jail, "", "")
	require.NoError(t, env.Set("EXIST", "val"))

	assert.Equal(t, "val", std.GetDefault(env, "EXIST", "other"))

	// empty value should fall back to provided default
	require.NoError(t, env.Set("EMPTY", ""))
	assert.Equal(t, "def", std.GetDefault(env, "EMPTY", "def"))

	// missing key should return fallback
	assert.Equal(t, "fallback", std.GetDefault(env, "MISSING", "fallback"))
}

// Ensure changing the test MapEnv does not modify the real process environment.
func TestTestEnvDoesNotChangeOsEnv(t *testing.T) {
	const key = "GO_STD_TEST_OS_ENV_KEY"

	// Preserve original OS env value and restore on exit.
	orig, ok := os.LookupEnv(key)
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, orig)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	// Set a known value in the real OS environment.
	require.NoError(t, os.Setenv(key, "os-value"))

	// Create a test env and change the same key in the MapEnv.
	jail := t.TempDir()
	env := std.NewTestEnv(jail, "", "")
	require.NoError(t, env.Set(key, "test-value"))

	// The real OS environment should remain unchanged.
	assert.Equal(t, "os-value", os.Getenv(key))

	// Unsetting in the MapEnv should not affect the real OS env either.
	env.Unset(key)
	assert.Equal(t, "os-value", os.Getenv(key))
}

func TestExpandEnv(t *testing.T) {
	jail := t.TempDir()
	// Do not run this test in parallel because it temporarily sets real OS env.
	env := std.NewTestEnv(jail, "", "")
	require.NoError(t, env.Set("FOO", "bar"))
	require.NoError(t, env.Set("EMPTY", ""))

	ctx := std.WithEnv(context.Background(), env)

	// Simple $VAR expansion
	got := std.ExpandEnv(ctx, "$FOO/baz")
	assert.Equal(t, filepath.Join("bar", "baz"), got)

	// Braced form, missing and empty values
	got2 := std.ExpandEnv(ctx, "${FOO}_${MISSING}_${EMPTY}")
	assert.Equal(t, "bar__", got2)

	// When no Env is provided in the context, ExpandEnv should fall back to the
	// real OS environment (OsEnv). Verify by setting a real OS env var.
	const oskey = "GO_STD_TEST_EXPAND_ENV_OS"
	orig, ok := os.LookupEnv(oskey)
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(oskey, orig)
		} else {
			_ = os.Unsetenv(oskey)
		}
	})
	require.NoError(t, os.Setenv(oskey, "osval"))

	got3 := std.ExpandEnv(context.Background(), "$"+oskey)
	assert.Equal(t, "osval", got3)
}
