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
// (MapEnv). The interface intentionally mirrors common environment
// operations so callers can easily swap a test implementation in unit
// tests without touching the real process environment.
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

	// GetWd returns the working directory as seen by this Env. For OsEnv this
	// is the process working directory; for MapEnv it is the stored PWD.
	GetWd() (string, error)

	// SetWd sets the working directory for this Env. For OsEnv this may
	// change the process working directory; for MapEnv it updates the stored PWD.
	SetWd(dir string)

	// GetTempDir returns an appropriate temp directory for this Env. For OsEnv
	// this delegates to os.TempDir(); MapEnv provides testable fallbacks.
	GetTempDir() string
}

// OsEnv implements Env by delegating to the real process environment.
// Use this in production code.
type OsEnv struct{}

// GetHome returns the OS-reported user home directory.
func (o *OsEnv) GetHome() (string, error) {
	return os.UserHomeDir()
}

// SetHome sets environment values that represent the user's home directory.
// On Unix-like systems this sets HOME. On Windows it also sets USERPROFILE to
// keep common consumers satisfied.
func (o *OsEnv) SetHome(home string) error {
	if runtime.GOOS == "windows" {
		if err := os.Setenv("USERPROFILE", home); err != nil {
			return err
		}
	}
	return os.Setenv("HOME", home)
}

// GetUser returns the current OS user username. If Username is empty it falls
// back to the user's Name field.
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

// SetUser sets environment values that represent the current user.
// On Unix-like systems this sets USER. On Windows it also sets USERNAME.
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

// Set sets the OS environment variable.
func (o *OsEnv) Set(key string, value string) error {
	return os.Setenv(key, value)
}

// Unset removes the OS environment variable.
func (o *OsEnv) Unset(key string) {
	os.Unsetenv(key)
}

// GetTempDir returns the OS temporary directory.
func (o *OsEnv) GetTempDir() string {
	return os.TempDir()
}

// GetWd returns the current process working directory.
func (o *OsEnv) GetWd() (string, error) {
	return os.Getwd()
}

// SetWd attempts to change the process working directory to dir and also sets
// the PWD environment variable. Errors from chdir are intentionally ignored by
// this method; callers who need to observe chdir errors can perform os.Chdir
// directly.
func (o *OsEnv) SetWd(dir string) {
	_ = os.Setenv("PWD", dir)
	_ = os.Chdir(dir)
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

// Get returns the stored value for key. Reading from a nil map returns the
// zero value, so this method is safe on a zero MapEnv. The special keys HOME
// and USER come from dedicated fields.
func (m MapEnv) Get(key string) string {
	switch key {
	case "HOME":
		return m.home
	case "USER":
		return m.user
	default:
		return m.data[key]
	}
}

// GetHome returns the configured home directory or an error if it is not set.
func (m *MapEnv) GetHome() (string, error) {
	if m == nil || m.home == "" {
		return "", errors.New("home not set in MapEnv")
	}
	return m.home, nil
}

// SetHome sets the MapEnv's home directory and updates the "HOME" key in
// the underlying map for callers that read via Get.
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

// GetUser returns the configured username or an error if it is not set.
func (m *MapEnv) GetUser() (string, error) {
	if m == nil || m.user == "" {
		return "", errors.New("user not set in MapEnv")
	}
	return m.user, nil
}

// SetUser sets the MapEnv's current user and updates the "USER" key in the
// underlying map for callers that use Get.
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

// Set stores a key/value pair in the MapEnv. If key is "HOME" or "USER" the
// corresponding dedicated field is updated. Calling Set on a nil receiver
// returns an error rather than panicking.
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

// Unset removes a key from the MapEnv. If key is "HOME" or "USER" the
// corresponding field is cleared. Calling Unset on a nil receiver is a no-op.
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

// GetTempDir returns a temp directory appropriate for the MapEnv. If the
// receiver is nil this falls back to os.TempDir to avoid panics.
//
// The method prefers explicit TMPDIR/TEMP/TMP values stored in the MapEnv.
// On Windows it applies a series of fallbacks: LOCALAPPDATA/APPDATA/USERPROFILE,
// then a home-based default. On Unix-like systems it falls back to /tmp.
func (m *MapEnv) GetTempDir() string {
	if m == nil {
		return os.TempDir()
	}

	// Prefer explicit TMPDIR/TEMP/TMP if provided in the MapEnv.
	if d := m.data["TMPDIR"]; d != "" {
		return d
	}
	if d := m.data["TEMP"]; d != "" {
		return d
	}
	if d := m.data["TMP"]; d != "" {
		return d
	}

	// Platform-specific sensible defaults without consulting the real process env.
	if runtime.GOOS == "windows" {
		// Prefer LOCALAPPDATA, then APPDATA, then USERPROFILE, then a home-based default.
		if local := m.data["LOCALAPPDATA"]; local != "" {
			return filepath.Join(local, "Temp")
		}
		if app := m.data["APPDATA"]; app != "" {
			return filepath.Join(app, "Temp")
		}
		if up := m.data["USERPROFILE"]; up != "" {
			return filepath.Join(up, "Temp")
		}
		if m.home != "" {
			return filepath.Join(m.home, "AppData", "Local", "Temp")
		}
		// No information available in MapEnv; return empty string to indicate unknown.
		return ""
	}

	// Unix-like: fall back to /tmp which is the conventional system temp dir.
	return "/tmp"
}

// GetWd returns the MapEnv's PWD value if set, otherwise an error.
func (m *MapEnv) GetWd() (string, error) {
	if m == nil {
		return "", errors.New("wd not set in MapEnv")
	}
	if m.data != nil {
		if wd := m.data["PWD"]; wd != "" {
			return wd, nil
		}
	}
	return "", errors.New("wd not set in MapEnv")
}

// SetWd sets the MapEnv's PWD value to the provided directory. Calling
// SetWd on a nil receiver is a no-op.
func (m *MapEnv) SetWd(dir string) {
	if m == nil {
		return
	}
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data["PWD"] = dir
}

// NewTestEnv constructs a MapEnv populated with sensible defaults for tests.
// It sets HOME and USER and also sets platform-specific variables so functions
// that prefer XDG_* (Unix) or APPDATA/LOCALAPPDATA (Windows) will pick them up.
//
// If home or username are empty, reasonable defaults are chosen:
//   - home defaults to os.TempDir()/home/<username>
//   - username defaults to "testuser"
//
// The function does not create any directories on disk; it only sets the
// environment values in the returned MapEnv.
func NewTestEnv(home, username string) *MapEnv {
	basepath := os.TempDir()
	if username == "" {
		username = "testuser"
	}
	if home == "" {
		home = filepath.Join(basepath, "home", username)
	}

	m := &MapEnv{
		home: home,
		user: username,
		data: make(map[string]string),
	}

	// Always expose HOME and USER through the map as well for callers that read via Get.
	m.data["HOME"] = home
	m.data["USER"] = username

	m.data["PWD"] = basepath

	// Populate platform-specific defaults so callers that query these keys get
	// consistent results in tests.
	if runtime.GOOS == "windows" {
		// Windows conventions: APPDATA (Roaming) and LOCALAPPDATA (Local)
		appdata := filepath.Join(home, "AppData", "Roaming")
		local := filepath.Join(home, "AppData", "Local")
		m.data["APPDATA"] = appdata
		m.data["LOCALAPPDATA"] = local
		m.data["TMPDIR"] = filepath.Join(local, "Temp")
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
		m.data["TMPDIR"] = filepath.Join(basepath, "tmp")
	}

	return m
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

// Ensure implementations satisfy the interfaces.
var _ Env = (*OsEnv)(nil)
var _ Env = (*MapEnv)(nil)

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
