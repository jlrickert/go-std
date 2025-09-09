package std_test

import (
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
