package std

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
)

// Package std provides helpers for locating user-specific directories
// (config, cache, data, state) in a cross-platform, testable way.
// The functions prefer platform-specific environment variables (XDG_* on
// Unix-like systems, APPDATA/LOCALAPPDATA on Windows) and fall back to
// sensible defaults under the user's home directory when those are not set.
//
// Note: These helpers only compute and return paths; they do not create
// directories on disk.
//
// The env parameter (of type Env) is used to read environment variables
// in a way that's easy to mock for testing. Some functions may still read
// certain variables directly from the real environment when appropriate.

// UserConfigPath returns the directory that should be used to store
// per-user configuration files.
//
// Behavior:
//   - On Unix-like systems: prefers XDG_CONFIG_HOME if set; otherwise falls
//     back to $HOME/.config.
//   - On Windows: prefers APPDATA (via the provided env) if set.
//   - The function returns the raw directory value as used by the platform.
//     It does not always append appName — callers should append the
//     application-specific subdirectory if desired.
func UserConfigPath(ctx context.Context) (string, error) {
	env := EnvFromContext(ctx)
	// Prefer explicit XDG_CONFIG_HOME on Unix-like systems (read from real env)
	if xdg := env.Get("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}
	// Prefer APPDATA on Windows (read via injected env for testability)
	if app := env.Get("APPDATA"); app != "" {
		return app, nil
	}
	home, err := env.GetHome()
	if err != nil {
		return "", err
	}
	// Fallback to the conventional Unix-style config directory.
	return filepath.Join(home, ".config"), nil
}

// UserCachePath returns the directory that should be used to store
// per-user cache files.
//
// Behavior:
// - On Unix-like systems: prefers XDG_CACHE_HOME if set.
// - On Windows: prefers LOCALAPPDATA if set.
// - Falls back to $HOME/.cache when no appropriate environment variable is set.
//
// As with UserConfigPath, the returned path is the base cache directory;
// append appName if you want an application-specific subdirectory.
func UserCachePath(ctx context.Context, appName string) (string, error) {
	env := EnvFromContext(ctx)
	if x := env.Get("XDG_CACHE_HOME"); x != "" {
		return x, nil
	}
	if local := env.Get("LOCALAPPDATA"); local != "" {
		return local, nil
	}
	home, err := env.GetHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache"), nil
}

// UserDataPath returns the directory that should be used to store
// per-user application data.
//
// Behavior:
//   - On Windows: uses LOCALAPPDATA and appends appName and "data".
//     If LOCALAPPDATA is not set, returns an error wrapping ErrNoEnvKey.
//   - On Unix-like systems: prefers XDG_DATA_HOME if set (appName appended),
//     otherwise falls back to ~/.local/share/<appName>.
func UserDataPath(ctx context.Context) (string, error) {
	env := EnvFromContext(ctx)
	if runtime.GOOS == "windows" {
		if localAppData := env.Get("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "data"), nil
		}
		// ErrNoEnvKey should be defined elsewhere in this package to indicate
		// that a required environment key was not present.
		return "", fmt.Errorf("LOCALAPPDATA environment variable not set: %w", ErrNoEnvKey)
	}
	// Unix-like: use $XDG_DATA_HOME if available, otherwise fallback to ~/.local/share
	if xdg := env.Get("XDG_DATA_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := env.GetHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

// UserStatePath returns the directory that should be used to store
// per-user state files for an application (transient runtime state).
//
// Behavior:
//   - On Windows: uses LOCALAPPDATA and appends appName and "state".
//     If LOCALAPPDATA is not set, returns an error wrapping ErrNoEnvKey.
//   - On Unix-like systems: prefers XDG_STATE_HOME if set (appName appended),
//     otherwise falls back to ~/.local/state/<appName>.
func UserStatePath(ctx context.Context) (string, error) {
	env := EnvFromContext(ctx)
	if runtime.GOOS == "windows" {
		if localAppData := env.Get("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "state"), nil
		}
		return "", fmt.Errorf("LOCALAPPDATA environment variable not set: %w", ErrNoEnvKey)
	}
	// Unix-like: use $XDG_STATE_HOME if available, otherwise fallback to ~/.local/state
	if xdg := env.Get("XDG_STATE_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := env.GetHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state"), nil
}
