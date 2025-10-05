package std

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Package std provides helpers for locating user-specific directories
// (config, cache, data, state) in a cross-platform, testable way.
// The functions prefer platform-specific environment variables
// (XDG_* on Unix-like systems, APPDATA/LOCALAPPDATA on Windows) and fall
// back to sensible defaults under the user's home directory when those are
// not set.
//
// Note: These helpers only compute and return paths; they do not create
// directories on disk.
//
// The env parameter (of type Env) is used to read environment variables
// in a way that is easy to mock for testing. Some functions may still read
// certain variables directly from the real environment when appropriate.

// ExpandPath expands a leading tilde in the provided path to the user's
// home directory obtained from the Env stored in ctx. Supported forms:
//
//	"~"
//	"~/rest/of/path"
//	"~\rest\of\path" (Windows)
//
// If the home directory cannot be obtained from the environment an error is
// returned. If the path does not start with a tilde it is returned unchanged.
func ExpandPath(ctx context.Context, p string) (string, error) {
	if p == "" {
		return p, nil
	}
	if p[0] != '~' {
		return p, nil
	}

	// Only expand the simple leading-tilde forms: "~" or "~/" or "~\"
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		env := EnvFromContext(ctx)
		home, err := env.GetHome()
		if err != nil {
			return "", err
		}
		if p == "~" {
			return filepath.Clean(home), nil
		}
		// Trim the "~/" or "~\" prefix and join with home to produce a
		// well-formed path.
		rest := p[2:]
		return filepath.Join(home, rest), nil
	}

	// More complex cases like "~username/..." are not supported and are
	// returned unchanged.
	return p, nil
}

// UserConfigPath returns the directory that should be used to store
// per-user configuration files.
//
// Behavior:
//   - On Unix-like systems: prefers XDG_CONFIG_HOME if set; otherwise falls
//     back to $HOME/.config.
//   - On Windows: prefers APPDATA (via the provided env) if set.
//
// The function returns the raw directory value as used by the platform.
// Callers should append an application-specific subdirectory if desired.
func UserConfigPath(ctx context.Context) (string, error) {
	env := EnvFromContext(ctx)
	// Prefer explicit XDG_CONFIG_HOME on Unix-like systems
	if xdg := env.Get("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Clean(xdg), nil
	}
	// Prefer APPDATA on Windows (read via injected env for testability)
	if app := env.Get("APPDATA"); app != "" {
		return filepath.Clean(app), nil
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
//   - On Unix-like systems: prefers XDG_CACHE_HOME if set.
//   - On Windows: prefers LOCALAPPDATA if set.
//   - Falls back to $HOME/.cache when no appropriate environment variable is
//     set.
//
// The returned path is the base cache directory; append appName if you want
// an application-specific subdirectory.
func UserCachePath(ctx context.Context) (string, error) {
	env := EnvFromContext(ctx)
	if xdg := env.Get("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Clean(xdg), nil
	}
	if local := env.Get("LOCALAPPDATA"); local != "" {
		return filepath.Clean(local), nil
	}
	home, err := env.GetHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache"), nil
}

// UserDataPath returns the directory that should be used to store per-user
// application data.
//
// Behavior:
//   - On Windows: uses LOCALAPPDATA and returns a "data" directory under it.
//     If LOCALAPPDATA is not set, the function returns an error that wraps
//     ErrNoEnvKey.
//   - On Unix-like systems: prefers XDG_DATA_HOME if set; otherwise falls back
//     to ~/.local/share.
//
// Callers should append an application-specific subdirectory if desired.
func UserDataPath(ctx context.Context) (string, error) {
	env := EnvFromContext(ctx)
	if runtime.GOOS == "windows" {
		if localAppData := env.Get("LOCALAPPDATA"); localAppData != "" {
			return filepath.Clean(filepath.Join(localAppData, "data")), nil
		}
		return "", fmt.Errorf(
			"LOCALAPPDATA environment variable not set: %w",
			ErrNoEnvKey,
		)
	}
	// Unix-like: use $XDG_DATA_HOME if available, otherwise fallback to
	// ~/.local/share
	if xdg := env.Get("XDG_DATA_HOME"); xdg != "" {
		return filepath.Clean(xdg), nil
	}
	home, err := env.GetHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

// UserStatePath returns the directory that should be used to store per-user
// state files for an application (transient runtime state).
//
// Behavior:
//   - On Windows: uses LOCALAPPDATA and returns a "state" directory under it.
//     If LOCALAPPDATA is not set, the function returns an error that wraps
//     ErrNoEnvKey.
//   - On Unix-like systems: prefers XDG_STATE_HOME if set; otherwise falls back
//     to ~/.local/state.
//
// Callers should append an application-specific subdirectory if desired.
func UserStatePath(ctx context.Context) (string, error) {
	env := EnvFromContext(ctx)
	if runtime.GOOS == "windows" {
		if localAppData := env.Get("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "state"), nil
		}
		return "", fmt.Errorf(
			"LOCALAPPDATA environment variable not set: %w",
			ErrNoEnvKey,
		)
	}
	// Unix-like: use $XDG_STATE_HOME if available, otherwise fallback to
	// ~/.local/state
	if xdg := env.Get("XDG_STATE_HOME"); xdg != "" {
		return xdg, nil
	}
	home, err := env.GetHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state"), nil
}

var DefaultEditor = "nano"

// Edit launches the user's editor to edit the provided file path.
// It checks $VISUAL first, then $EDITOR. If neither is set, it falls back to
// "nano". The function attaches the current process's stdio to the editor so
// interactive editors work as expected.
func Edit(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("empty filepath")
	}

	editor := os.Getenv("VISUAL")
	if strings.TrimSpace(editor) == "" {
		editor = os.Getenv("EDITOR")
	}
	if strings.TrimSpace(editor) == "" {
		editor = DefaultEditor
	}

	parts := strings.Fields(editor)
	name := parts[0]
	args := append(parts[1:], path)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running editor %q: %w", editor, err)
	}
	return nil
}
