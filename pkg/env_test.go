package std_test

import (
	"os"
	"testing"

	"github.com/jlrickert/go-std/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefault(t *testing.T) {
	env := std.NewTestEnv("", "")
	require.NoError(t, env.Set("EXIST", "val"))

	assert.Equal(t, "val", std.GetDefault(env, "EXIST", "other"))

	// empty value should fall back to provided default
	require.NoError(t, env.Set("EMPTY", ""))
	assert.Equal(t, "def", std.GetDefault(env, "EMPTY", "def"))

	// missing key should return fallback
	assert.Equal(t, "fallback", std.GetDefault(env, "MISSING", "fallback"))
}

func TestMapEnvSetUnsetHomeUser(t *testing.T) {
	m := std.NewTestEnv("/foo/home", "alice")

	home, err := m.GetHome()
	require.NoError(t, err)
	assert.Equal(t, "/foo/home", home)

	require.NoError(t, m.Set("HOME", "/bar"))
	home, err = m.GetHome()
	require.NoError(t, err)
	assert.Equal(t, "/bar", home)

	m.Unset("HOME")
	_, err = m.GetHome()
	require.Error(t, err)
}

// Ensure changing the test MapEnv does not modify the real process environment.
func TestMapEnvDoesNotChangeOsEnv(t *testing.T) {
	const key = "GO_STD_TEST_OS_ENV_KEY"

	// Preserve original OS env value and restore on exit.
	orig, ok := os.LookupEnv(key)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	})

	// Set a known value in the real OS environment.
	require.NoError(t, os.Setenv(key, "os-value"))

	// Create a test env and change the same key in the MapEnv.
	env := std.NewTestEnv("", "")
	require.NoError(t, env.Set(key, "test-value"))

	// The real OS environment should remain unchanged.
	assert.Equal(t, "os-value", os.Getenv(key))

	// Unsetting in the MapEnv should not affect the real OS env either.
	env.Unset(key)
	assert.Equal(t, "os-value", os.Getenv(key))
}
