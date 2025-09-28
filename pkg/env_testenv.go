package std

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// TestEnv is an in-memory Env implementation useful for tests. It does not
// touch the real process environment and therefore makes tests hermetic.
//
// The home and user fields are used to satisfy GetHome/GetUser. The map
// stores all other keys. For convenience, setting or unsetting the keys
// "HOME" and "USER" will update the corresponding home/user fields.
type TestEnv struct {
	jail string
	home string
	user string
	data map[string]string
}

// Ensure implementations satisfy the interfaces.
var _ Env = (*TestEnv)(nil)

// GetHome returns the configured home directory or an error if it is not set.
//
// For TestEnv the returned home is guaranteed to be located within the
// configured jail when possible. This helps keep tests hermetic by ensuring
// paths used for home are under the test temporary area.
func (m *TestEnv) GetHome() (string, error) {
	if m == nil || m.home == "" {
		return "", errors.New("home not set in MapEnv")
	}
	return EnsureInJail(m.jail, m.home), nil
}

// SetHome sets the MapEnv's home directory and updates the "HOME" key in
// the underlying map for callers that read via Get.
func (m *TestEnv) SetHome(home string) error {
	if m == nil {
		return errors.New("nil MapEnv")
	}
	m.home = EnsureInJail(m.jail, home)
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data["HOME"] = m.home
	return nil
}

// GetUser returns the configured username or an error if it is not set.
func (m *TestEnv) GetUser() (string, error) {
	if m == nil || m.user == "" {
		return "", errors.New("user not set in MapEnv")
	}
	return m.user, nil
}

// SetUser sets the MapEnv's current user and updates the "USER" key in the
// underlying map for callers that use Get.
func (m *TestEnv) SetUser(username string) error {
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

// Get returns the stored value for key. Reading from a nil map returns the
// zero value, so this method is safe on a zero MapEnv. The special keys HOME
// and USER come from dedicated fields.
func (m TestEnv) Get(key string) string {
	switch key {
	case "HOME":
		return m.home
	case "USER":
		return m.user
	default:
		return m.data[key]
	}
}

// Set stores a key/value pair in the MapEnv. If key is "HOME" or "USER" the
// corresponding dedicated field is updated. Calling Set on a nil receiver
// returns an error rather than panicking.
func (m *TestEnv) Set(key string, value string) error {
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

func (m *TestEnv) Has(key string) bool {
	_, ok := m.data[key]
	return ok
}

// Unset removes a key from the MapEnv. If key is "HOME" or "USER" the
// corresponding field is cleared. Calling Unset on a nil receiver is a no-op.
func (m *TestEnv) Unset(key string) {
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
//
// The returned path will be adjusted to reside inside the configured jail
// when possible to keep test artifacts contained.
func (m *TestEnv) GetTempDir() string {
	if m == nil {
		return os.TempDir()
	}

	// Prefer explicit TMPDIR/TEMP/TMP if provided in the MapEnv.
	if d := m.data["TMPDIR"]; d != "" {
		return EnsureInJail(m.jail, d)
	}
	if d := m.data["TEMP"]; d != "" {
		return EnsureInJail(m.jail, d)
	}
	if d := m.data["TMP"]; d != "" {
		return EnsureInJail(m.jail, d)
	}

	// Platform-specific sensible defaults without consulting the real process env.
	if runtime.GOOS == "windows" {
		// Prefer LOCALAPPDATA, then APPDATA, then USERPROFILE, then a home-based default.
		if local := m.data["LOCALAPPDATA"]; local != "" {
			return EnsureInJail(m.jail, filepath.Join(local, "Temp"))
		}
		if app := m.data["APPDATA"]; app != "" {
			return EnsureInJail(m.jail, filepath.Join(app, "Temp"))
		}
		if up := m.data["USERPROFILE"]; up != "" {
			return EnsureInJail(m.jail, filepath.Join(up, "Temp"))
		}
		if m.home != "" {
			return EnsureInJail(m.jail,
				filepath.Join(m.home, "AppData", "Local", "Temp"))
		}
		// No information available in MapEnv; return empty string to indicate unknown.
		return ""
	}

	// Unix-like: fall back to /tmp which is the conventional system temp dir.
	return EnsureInJail(m.jail, "/tmp")
}

// GetWd returns the MapEnv's PWD value if set, otherwise an error.
func (m *TestEnv) Getwd() (string, error) {
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
func (m *TestEnv) Setwd(dir string) {
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
func NewTestEnv(jail, home, username string) *TestEnv {
	if username == "" {
		username = "testuser"
	}
	if home == "" {
		home = EnsureInJail(jail, home)
	}

	m := &TestEnv{
		jail: jail,
		home: home,
		user: username,
		data: make(map[string]string),
	}

	// Always expose HOME and USER through the map as well for callers that read via Get.
	m.data["HOME"] = home
	m.data["USER"] = username

	m.data["PWD"] = jail

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
		m.data["TMPDIR"] = filepath.Join(jail, "tmp")
	}

	return m
}

func (m *TestEnv) Mkdir(path string, perm os.FileMode, all bool) error {
	if all {
		return os.MkdirAll(EnsureInJail(m.jail, path), perm)
	}
	return os.Mkdir(EnsureInJail(m.jail, path), perm)
}

func (m *TestEnv) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(EnsureInJail(m.jail, name), data, perm)
}

func (m *TestEnv) Root() string {
	return m.jail
}
