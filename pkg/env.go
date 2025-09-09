package std

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// Env is a small, descriptive interface for reading and modifying
// environment values. Implementations may read from the real process
// environment (OsEnv) or from an in-memory store useful for tests
// (MapEnv).
type Env interface {
	// Get returns the raw environment value for key (may be empty).
	Get(key string) string

	// Set assigns the environment key to value.
	Set(key, value string) error

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
}

// OsEnv implements Env by delegating to the real process environment.
// Use this in production code.
type OsEnv struct{}

// GetHome implements Env by returning the OS-reported user home directory.
func (o *OsEnv) GetHome() (string, error) {
	return os.UserHomeDir()
}

// SetHome implements Env by setting environment values that represent the
// user's home directory. On Unix-like systems this sets HOME. On Windows it
// sets HOME and USERPROFILE to keep common consumers satisfied.
func (o *OsEnv) SetHome(home string) error {
	if runtime.GOOS == "windows" {
		if err := os.Setenv("USERPROFILE", home); err != nil {
			return err
		}
	}
	return os.Setenv("HOME", home)
}

// GetUser implements Env by returning the current OS user username.
func (o *OsEnv) GetUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	// Username is typically what callers expect; fall back to Name if empty.
	if u.Username != "" {
		return u.Username, nil
	}
	return u.Name, nil
}

// SetUser implements Env by setting environment values that represent the
// current user. On Unix-like systems this sets USER. On Windows it sets both
// USER and USERNAME to maximize compatibility.
func (o *OsEnv) SetUser(username string) error {
	if runtime.GOOS == "windows" {
		if err := os.Setenv("USERNAME", username); err != nil {
			return err
		}
	}
	return os.Setenv("USER", username)
}

// Get returns the environment variable for key.
func (o *OsEnv) Get(key string) string {
	return os.Getenv(key)
}

// Set implements Env by setting the OS environment variable.
func (o *OsEnv) Set(key string, value string) error {
	return os.Setenv(key, value)
}

// Unset implements Env by unsetting the OS environment variable.
func (o *OsEnv) Unset(key string) {
	os.Unsetenv(key)
}

// MapEnv is an in-memory Env implementation useful for tests. It does not
// touch the real process environment and therefore makes tests hermetic.
//
// The home and user fields are used to satisfy GetHome/GetUser. The map
// stores all other keys. For convenience, setting or unsetting the keys
// "HOME" and "USER" will update the corresponding home/user fields.
type MapEnv struct {
	home string
	user string
	data map[string]string
}

// Get implements Env for MapEnv.
func (m MapEnv) Get(key string) string {
	// Reading from a nil map returns the zero value, so this is safe.
	// Treat the special keys HOME and USER as coming from the dedicated fields.
	switch key {
	case "HOME":
		return m.home
	case "USER":
		return m.user
	default:
		return m.data[key]
	}
}

// GetHome implements Env. Returns an error if home is not set.
func (m *MapEnv) GetHome() (string, error) {
	if m == nil || m.home == "" {
		return "", errors.New("home not set in MapEnv")
	}
	return m.home, nil
}

// SetHome sets the MapEnv's home directory and updates the "HOME" key in
// the underlying map for callers that use Get.
func (m *MapEnv) SetHome(home string) error {
	if m == nil {
		return errors.New("nil MapEnv")
	}
	m.home = home
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data["HOME"] = home
	return nil
}

// GetUser implements Env. Returns an error if user is not set.
func (m *MapEnv) GetUser() (string, error) {
	if m == nil || m.user == "" {
		return "", errors.New("user not set in MapEnv")
	}
	return m.user, nil
}

// SetUser sets the MapEnv's current user and updates the "USER" key in
// the underlying map for callers that use Get.
func (m *MapEnv) SetUser(username string) error {
	if m == nil {
		return errors.New("nil MapEnv")
	}
	m.user = username
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data["USER"] = username
	return nil
}

// Set implements Env. If key is "HOME" or "USER" the corresponding field
// is updated, otherwise the value is stored in the internal map.
func (m *MapEnv) Set(key string, value string) error {
	if m == nil {
		// Preserve the original behavior of making a new MapEnv when Set is
		// called on a nil receiver. However, since we cannot mutate the
		// caller's nil pointer here, return an error to indicate misuse.
		return errors.New("nil MapEnv")
	}
	switch key {
	case "HOME":
		return m.SetHome(value)
	case "USER":
		return m.SetUser(value)
	default:
		if m.data == nil {
			m.data = make(map[string]string)
		}
		m.data[key] = value
	}
	return nil
}

// Unset implements Env. If key is "HOME" or "USER" the corresponding field
// is cleared, otherwise the key is removed from the internal map.
func (m *MapEnv) Unset(key string) {
	if m == nil {
		return
	}
	switch key {
	case "HOME":
		m.home = ""
		if m.data != nil {
			delete(m.data, "HOME")
		}
	case "USER":
		m.user = ""
		if m.data != nil {
			delete(m.data, "USER")
		}
	default:
		if m.data != nil {
			delete(m.data, key)
		}
	}
}

// NewTestEnv constructs a MapEnv populated with sensible defaults for tests.
// It sets HOME and USER and also sets platform-specific variables so functions
// that prefer XDG_* (Unix) or APPDATA/LOCALAPPDATA (Windows) will pick them up.
//
// If home or username are empty, reasonable defaults are chosen:
//   - home defaults to os.TempDir()/go-std-test-home
//   - username defaults to "testuser"
//
// The function does not create any directories on disk; it only sets the
// environment values in the returned MapEnv.
func NewTestEnv(home, username string) *MapEnv {
	if username == "" {
		username = "testuser"
	}
	if home == "" {
		home = filepath.Join(os.TempDir(), "home", username)
	}

	m := &MapEnv{
		home: home,
		user: username,
		data: make(map[string]string),
	}

	// Always expose HOME and USER through the map as well for callers that read via Get.
	m.data["HOME"] = home
	m.data["USER"] = username

	if runtime.GOOS == "windows" {
		// Windows conventions: APPDATA (Roaming) and LOCALAPPDATA (Local)
		appdata := filepath.Join(home, "AppData", "Roaming")
		local := filepath.Join(home, "AppData", "Local")
		m.data["APPDATA"] = appdata
		m.data["LOCALAPPDATA"] = local
	} else {
		// Unix-like conventions: XDG_* fallbacks under the home directory.
		xdgConfig := filepath.Join(home, ".config")
		xdgCache := filepath.Join(home, ".cache")
		xdgData := filepath.Join(home, ".local", "share")
		xdgState := filepath.Join(home, ".local", "state")
		m.data["XDG_CONFIG_HOME"] = xdgConfig
		m.data["XDG_CACHE_HOME"] = xdgCache
		m.data["XDG_DATA_HOME"] = xdgData
		m.data["XDG_STATE_HOME"] = xdgState
	}

	return m
}

// GetDefault returns the value of key from the provided Env if present
// (non-empty), otherwise it returns the provided fallback value.
// Use this helper to prefer an environment value while allowing a default.
func GetDefault(env Env, key, other string) string {
	if env == nil {
		return other
	}
	if v := env.Get(key); v != "" {
		return v
	}
	return other
}

func ExpandEnv(env Env, s string) string {
	if env == nil {
		env = &OsEnv{}
	}
	return os.Expand(s, env.Get)
}

// Ensure implementations satisfy the interfaces.
var _ Env = (*OsEnv)(nil)
var _ Env = (*MapEnv)(nil)

var ctxEnvKey ctxKeyType

func ContextWithEnv(ctx context.Context, env Env) context.Context {
	return context.WithValue(ctx, ctxEnvKey, env)
}

func EnvFromContext(ctx context.Context) Env {
	if ctx == nil {
		return &OsEnv{}
	}
	if v := ctx.Value(ctxEnvKey); v != nil {
		if env, ok := v.(Env); ok && env != nil {
			return env
		}
	}
	return &OsEnv{}
}
