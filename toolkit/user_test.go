package toolkit_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserConfigPath(t *testing.T) {
	// Prefer XDG_CONFIG_HOME when provided by the injected env.
	env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
	require.NoError(t, env.Set("XDG_CONFIG_HOME", "/real/xdg"))
	cfg, err := toolkit.UserConfigPath(toolkit.WithEnv(context.Background(), env))
	require.NoError(t, err)
	assert.Equal(t, "/real/xdg", cfg)

	// Clear XDG_CONFIG_HOME for following checks.
	env.Unset("XDG_CONFIG_HOME")

	// Platform-specific behavior: on Windows prefer APPDATA via env.
	if runtime.GOOS == "windows" {
		env := toolkit.NewTestEnv("", filepath.FromSlash("C:/Users/alice"), "alice")
		require.NoError(t, env.Set("APPDATA", filepath.FromSlash("C:/AppData")))
		cfg, err := toolkit.UserConfigPath(toolkit.WithEnv(context.Background(), env))
		require.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("C:/AppData"), cfg)
	} else {
		// Unix-like fallback to $HOME/.config.
		env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
		cfg, err := toolkit.UserConfigPath(toolkit.WithEnv(context.Background(), env))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("/home/alice", ".config"), cfg)

		// Error flows when home is not available.
		emptyEnv := &toolkit.TestEnv{} // no home set
		_, err = toolkit.UserConfigPath(toolkit.WithEnv(context.Background(), emptyEnv))
		require.Error(t, err)
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()
	t.Run("TildeOnly", func(t *testing.T) {
		jail := t.TempDir()
		relHome := filepath.Join("home", "alice")
		env := toolkit.NewTestEnv(jail, relHome, "alice")
		got, err := toolkit.ExpandPath(toolkit.WithEnv(t.Context(), env), "~")
		require.NoError(t, err)

		expected := toolkit.EnsureInJail(jail, relHome)
		assert.Equal(t, filepath.Clean(expected), filepath.Clean(got))
	})

	t.Run("TildeSlash", func(t *testing.T) {
		jail := t.TempDir()
		env := toolkit.NewTestEnv(jail, filepath.Join("home", "alice"), "alice")
		got, err := toolkit.ExpandPath(
			toolkit.WithEnv(t.Context(), env),
			filepath.Join("~", "project"),
		)
		require.NoError(t, err)

		expected := toolkit.EnsureInJail(jail, filepath.Join("home", "alice", "project"))
		assert.Equal(t, filepath.Clean(expected), filepath.Clean(got))
	})

	t.Run("NonTildeUnchanged", func(t *testing.T) {
		env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
		in := filepath.FromSlash("/tmp/some/path")
		got, err := toolkit.ExpandPath(toolkit.WithEnv(context.Background(), env), in)
		require.NoError(t, err)
		assert.Equal(t, in, got)
	})

	t.Run("EmptyString", func(t *testing.T) {
		env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
		got, err := toolkit.ExpandPath(toolkit.WithEnv(context.Background(), env), "")
		require.NoError(t, err)
		assert.Equal(t, "", got)
	})

	t.Run("UnsupportedUserFormReturnsUnchanged", func(t *testing.T) {
		env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
		in := "~bob/project"
		got, err := toolkit.ExpandPath(toolkit.WithEnv(context.Background(), env), in)
		require.NoError(t, err)
		assert.Equal(t, in, got)
	})

	t.Run("MissingHomeReturnsError", func(t *testing.T) {
		emptyEnv := &toolkit.TestEnv{}
		_, err := toolkit.ExpandPath(toolkit.WithEnv(context.Background(), emptyEnv), "~")
		require.Error(t, err)
	})

	// Platform-specific test for backslash-prefixed expansion on Windows.
	if runtime.GOOS == "windows" {
		t.Run("TildeBackslashWindows", func(t *testing.T) {
			home := filepath.FromSlash("C:/Users/alice")
			env := toolkit.NewTestEnv("", home, "alice")
			in := `~\project\sub`
			got, err := toolkit.ExpandPath(toolkit.WithEnv(context.Background(), env), in)
			require.NoError(t, err)
			// Expect the components to be joined using filepath semantics.
			assert.Equal(t, filepath.Join(home, "project", "sub"), got)
		})
	}
}

func TestUserCachePath(t *testing.T) {
	// XDG_CACHE_HOME provided by env should be returned.
	env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
	require.NoError(t, env.Set("XDG_CACHE_HOME", "/xdg/cache"))
	c, err := toolkit.UserCachePath(toolkit.WithEnv(context.Background(), env))
	require.NoError(t, err)
	assert.Equal(t, "/xdg/cache", c)

	// LOCALAPPDATA should be used on Windows when present.
	env = toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
	require.NoError(t, env.Set("LOCALAPPDATA", filepath.FromSlash("C:/Local")))
	c, err = toolkit.UserCachePath(toolkit.WithEnv(context.Background(), env))
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, filepath.FromSlash("C:/Local"), c)
	} else {
		// On Unix-like, LOCALAPPDATA is not considered first; fallback to home/.cache
		assert.Equal(t, filepath.Join("/home/alice", ".cache"), c)
	}

	// Fallback to home/.cache when no XDG_CACHE_HOME or LOCALAPPDATA.
	env = toolkit.NewTestEnv("", filepath.FromSlash("/home/bob"), "bob")
	c, err = toolkit.UserCachePath(toolkit.WithEnv(context.Background(), env))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/bob", ".cache"), c)

	// Error when home missing.
	emptyEnv := &toolkit.TestEnv{}
	_, err = toolkit.UserCachePath(toolkit.WithEnv(context.Background(), emptyEnv))
	require.Error(t, err)
}

func TestUserDataPath(t *testing.T) {
	// Platform-specific expectations.

	if runtime.GOOS == "windows" {
		// When LOCALAPPDATA present, return LOCALAPPDATA/data
		env := toolkit.NewTestEnv("", filepath.FromSlash("C:/Users/alice"), "alice")
		require.NoError(t, env.Set("LOCALAPPDATA", filepath.FromSlash("C:/LocalApp")))
		p, err := toolkit.UserDataPath(toolkit.WithEnv(context.Background(), env))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(filepath.FromSlash("C:/LocalApp"), "data"), p)

		// When LOCALAPPDATA missing, expect error wrapping ErrNoEnvKey
		env2 := toolkit.NewTestEnv("", filepath.FromSlash("C:/Users/alice"), "alice")
		env2.Unset("LOCALAPPDATA")
		_, err = toolkit.UserDataPath(toolkit.WithEnv(context.Background(), env2))
		require.Error(t, err)
		// error should wrap ErrNoEnvKey (string check is sufficient here)
		assert.Contains(t, err.Error(), toolkit.ErrNoEnvKey.Error())
	} else {
		// Unix-like: XDG_DATA_HOME from env.Get should be used.
		env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
		require.NoError(t, env.Set("XDG_DATA_HOME", "/xdg/data"))
		p, err := toolkit.UserDataPath(toolkit.WithEnv(context.Background(), env))
		require.NoError(t, err)
		assert.Equal(t, "/xdg/data", p)

		// Fallback to ~/.local/share
		env2 := toolkit.NewTestEnv("", filepath.FromSlash("/home/bob"), "bob")
		env2.Unset("XDG_DATA_HOME")
		p, err = toolkit.UserDataPath(toolkit.WithEnv(context.Background(), env2))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("/home/bob", ".local", "share"), p)

		// Error when home missing.
		emptyEnv := &toolkit.TestEnv{}
		_, err = toolkit.UserDataPath(toolkit.WithEnv(context.Background(), emptyEnv))
		require.Error(t, err)
	}
}

func TestUserStatePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		env := toolkit.NewTestEnv("", filepath.FromSlash("C:/Users/alice"), "alice")
		require.NoError(t, env.Set("LOCALAPPDATA", filepath.FromSlash("C:/LocalApp")))
		p, err := toolkit.UserStatePath(toolkit.WithEnv(context.Background(), env))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(filepath.FromSlash("C:/LocalApp"), "state"), p)

		// Missing LOCALAPPDATA -> error
		env2 := toolkit.NewTestEnv("", filepath.FromSlash("C:/Users/alice"), "alice")
		env2.Unset("LOCALAPPDATA")
		_, err = toolkit.UserStatePath(toolkit.WithEnv(context.Background(), env2))
		require.Error(t, err)
		assert.Contains(t, err.Error(), toolkit.ErrNoEnvKey.Error())
	} else {
		// XDG_STATE_HOME preferred when set
		env := toolkit.NewTestEnv("", filepath.FromSlash("/home/alice"), "alice")
		require.NoError(t, env.Set("XDG_STATE_HOME", "/xdg/state"))
		p, err := toolkit.UserStatePath(toolkit.WithEnv(context.Background(), env))
		require.NoError(t, err)
		assert.Equal(t, "/xdg/state", p)

		// Fallback to ~/.local/state
		env2 := toolkit.NewTestEnv("", filepath.FromSlash("/home/bob"), "bob")
		env2.Unset("XDG_STATE_HOME")
		p, err = toolkit.UserStatePath(toolkit.WithEnv(context.Background(), env2))
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("/home/bob", ".local", "state"), p)

		// Error when home missing.
		emptyEnv := &toolkit.TestEnv{}
		_, err = toolkit.UserStatePath(toolkit.WithEnv(context.Background(), emptyEnv))
		require.Error(t, err)
	}
}
