package std

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// TestEnv is an in-memory Env implementation useful for tests. It does not
// touch the real process environment and therefore makes tests hermetic.
//
// The home and user fields satisfy GetHome and GetUser. The data map stores
// other keys. For convenience, setting or unsetting the keys "HOME" and
// "USER" updates the corresponding home and user fields.
type TestEnv struct {
	jail string
	home string
	user string
	data map[string]string
}

// NewTestEnv constructs a TestEnv populated with sensible defaults for tests.
// It sets HOME and USER and also sets platform specific variables so functions
// that prefer XDG_* on Unix or APPDATA/LOCALAPPDATA on Windows will pick them
// up.
//
// If home or username are empty, reasonable defaults are chosen:
//   - home defaults to EnsureInJailFor(jail, "/home/<username>")
//   - username defaults to "testuser"
//
// The function does not create directories on disk. It only sets environment
// values in the returned TestEnv.
func NewTestEnv(jail, home, username string) *TestEnv {
	user := username
	if user == "" {
		user = "testuser"
	}

	if home == "" && user == "root" {
		home = EnsureInJailFor(jail, filepath.Join("/", ".root"))
	} else if home == "" {
		home = EnsureInJailFor(jail, filepath.Join("/", "home", user))
	} else {
		home = EnsureInJailFor(jail, home)
	}

	m := &TestEnv{
		jail: jail,
		home: home,
		user: username,
		data: make(map[string]string),
	}

	// Always expose HOME and USER through the map as well for callers that read
	// via Get.
	m.data["HOME"] = home
	m.data["USER"] = username
	m.data["PWD"] = home

	// Populate platform specific defaults so callers that query these keys get
	// consistent results in tests.
	if runtime.GOOS == "windows" {
		// Windows conventions: APPDATA (Roaming) and LOCALAPPDATA (Local).
		appdata := filepath.Join(home, "AppData", "Roaming")
		local := filepath.Join(home, "AppData", "Local")
		m.data["APPDATA"] = appdata
		m.data["LOCALAPPDATA"] = local
		m.data["TMPDIR"] = filepath.Join(local, "Temp")
	} else {
		// Unix like conventions: XDG_* fallbacks under the home directory.
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

// GetHome returns the configured home directory or an error if it is not set.
//
// For TestEnv the returned home is guaranteed to be located within the
// configured jail when possible. This helps keep tests hermetic by ensuring
// paths used for home are under the test temporary area.
func (m *TestEnv) GetHome() (string, error) {
	if m.home == "" {
		return "", errors.New("home not set in TestEnv")
	}
	return m.home, nil
}

// SetHome sets the TestEnv home directory and updates the "HOME" key in the
// underlying map for callers that read via Get.
func (m *TestEnv) SetHome(home string) error {
	m.home = EnsureInJailFor(m.jail, home)
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data["HOME"] = m.home
	return nil
}

// GetUser returns the configured username or an error if it is not set.
func (m *TestEnv) GetUser() (string, error) {
	if m.user == "" {
		return "", errors.New("user not set in TestEnv")
	}
	return m.user, nil
}

// SetUser sets the TestEnv current user and updates the "USER" key in the
// underlying map for callers that use Get.
func (m *TestEnv) SetUser(username string) error {
	m.user = username
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data["USER"] = username
	return nil
}

// Get returns the stored value for key. Reading from a nil map returns the
// zero value, so this method is safe on a zero TestEnv. The special keys HOME
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

// Set stores a key/value pair in the TestEnv. If key is "HOME" or "USER" the
// corresponding dedicated field is updated. Calling Set on a nil receiver
// returns an error rather than panicking.
func (m *TestEnv) Set(key string, value string) error {
	switch key {
	case "HOME":
		return m.SetHome(value)
	case "USER":
		return m.SetUser(value)
	case "PWD":
		m.Setwd(value)
		return nil
	default:
		if m.data == nil {
			m.data = make(map[string]string)
		}
		m.data[key] = value
	}
	return nil
}

// Environ returns a slice of "KEY=VALUE" entries representing the environment
// stored in the TestEnv. It guarantees HOME and USER are present when set.
func (m *TestEnv) Environ() []string {
	// Collect keys from the backing map and ensure HOME/USER are present
	// based on dedicated fields so callers get a complete view.
	keys := make([]string, 0, len(m.data)+2)
	seen := make(map[string]struct{}, len(m.data)+2)
	for k := range m.data {
		keys = append(keys, k)
		seen[k] = struct{}{}
	}
	if m.home != "" {
		if _, ok := seen["HOME"]; !ok {
			keys = append(keys, "HOME")
		}
	}
	if m.user != "" {
		if _, ok := seen["USER"]; !ok {
			keys = append(keys, "USER")
		}
	}

	sort.Strings(keys)

	out := make([]string, 0, len(keys))
	for _, k := range keys {
		var v string
		switch k {
		case "HOME":
			v = m.home
		case "USER":
			v = m.user
		default:
			v = m.data[k]
		}
		out = append(out, k+"="+v)
	}
	return out
}

// Has reports whether the given key is present in the TestEnv map.
func (m *TestEnv) Has(key string) bool {
	_, ok := m.data[key]
	return ok
}

// Unset removes a key from the TestEnv. If key is "HOME" or "USER" the
// corresponding field is cleared. Calling Unset on a nil receiver is a no-op.
func (m *TestEnv) Unset(key string) {
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

// GetTempDir returns a temp directory appropriate for the TestEnv. If the
// receiver is nil this falls back to os.TempDir to avoid panics.
//
// The method prefers explicit TMPDIR/TEMP/TMP values stored in the TestEnv.
// On Windows it applies a series of fallbacks: LOCALAPPDATA, APPDATA,
// USERPROFILE, then a home based default. On Unix like systems it falls back
// to /tmp.
//
// The returned path will be adjusted to reside inside the configured jail
// when possible to keep test artifacts contained.
func (m *TestEnv) GetTempDir() string {
	// Prefer explicit TMPDIR/TEMP/TMP if provided in the TestEnv.
	if d := m.data["TMPDIR"]; d != "" {
		return EnsureInJailFor(m.jail, d)
	}
	if d := m.data["TEMP"]; d != "" {
		return EnsureInJailFor(m.jail, d)
	}
	if d := m.data["TMP"]; d != "" {
		return EnsureInJailFor(m.jail, d)
	}

	// Platform specific sensible defaults without consulting the real process env.
	if runtime.GOOS == "windows" {
		// Prefer LOCALAPPDATA, then APPDATA, then USERPROFILE, then a home based
		// default.
		if local := m.data["LOCALAPPDATA"]; local != "" {
			return EnsureInJailFor(m.jail, filepath.Join(local, "Temp"))
		}
		if app := m.data["APPDATA"]; app != "" {
			return EnsureInJailFor(m.jail, filepath.Join(app, "Temp"))
		}
		if up := m.data["USERPROFILE"]; up != "" {
			return EnsureInJailFor(m.jail, filepath.Join(up, "Temp"))
		}
		if m.home != "" {
			return EnsureInJailFor(m.jail,
				filepath.Join(m.home, "AppData", "Local", "Temp"))
		}
		// No information available in TestEnv; return empty string to indicate
		// unknown.
		return ""
	}

	// Unix like: fall back to /tmp which is the conventional system temp dir.
	return EnsureInJailFor(m.jail, "/tmp")
}

// Getwd returns the TestEnv's PWD value if set, otherwise an error.
func (m *TestEnv) Getwd() (string, error) {
	if m.data != nil {
		if wd := m.data["PWD"]; wd != "" {
			return wd, nil
		}
	}
	return "", errors.New("wd not set in TestEnv")
}

// Setwd sets the TestEnv's PWD value to the provided directory.
func (m *TestEnv) Setwd(dir string) {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data["PWD"] = EnsureInJailFor(m.jail, m.ExpandPath(dir))
}

// ReadFile reads the named file from the filesystem view held by this TestEnv.
// When the receiver is nil the real filesystem is used.
func (m *TestEnv) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(EnsureInJail(m.jail, name))
}

// Remove removes the named file or directory. If all is true RemoveAll is used.
// When the receiver is nil the real filesystem is affected.
func (m *TestEnv) Remove(path string, all bool) error {
	p := EnsureInJail(m.jail, path)
	if all {
		return os.RemoveAll(p)
	}
	return os.Remove(p)
}

// Rename renames (moves) a file or directory. When the receiver is nil the
// operation is performed on the real filesystem.
func (m *TestEnv) Rename(src string, dst string) error {
	return os.Rename(EnsureInJail(m.jail, src), EnsureInJail(m.jail, dst))
}

// Mkdir creates a directory. If all is true MkdirAll is used.
func (m *TestEnv) Mkdir(path string, perm os.FileMode, all bool) error {
	p := EnsureInJailFor(m.jail, m.ExpandPath(path))
	if all {
		return os.MkdirAll(EnsureInJailFor(m.jail, p), perm)
	}
	return os.Mkdir(EnsureInJailFor(m.jail, path), perm)
}

// WriteFile writes data to a file in the filesystem view held by this TestEnv.
func (m *TestEnv) WriteFile(name string, data []byte, perm os.FileMode) error {
	path := EnsureInJail(m.jail, m.ExpandPath(name))
	return os.WriteFile(path, data, perm)
}

// Root returns the configured jail root for this TestEnv.
func (m *TestEnv) Root() string {
	return m.jail
}

// ExpandPath expands a leading tilde in the provided path to the TestEnv home.
// Supported forms:
//
//	"~"
//	"~/rest/of/path"
//	"~\rest\of\path" (Windows)
//
// If the path does not start with a tilde it is returned unchanged. This method
// uses the TestEnv GetHome value. If home is not set, expansion may produce
// an empty or unexpected result.
func (m *TestEnv) ExpandPath(p string) string {
	if p == "" {
		return p
	}
	if p[0] != '~' {
		return p
	}

	// Only expand the simple leading tilde forms: "~" or "~/" or "~\"
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, _ := m.GetHome()
		if p == "~" {
			return filepath.Clean(home)
		}
		// Trim the "~/" or "~\" prefix and join with home to produce a
		// well formed path.
		rest := p[2:]
		return filepath.Join(home, rest)
	}

	// More complex cases like "~username/..." are not supported and are
	// returned unchanged.
	return p
}

// Clone returns a copy of the TestEnv so tests can modify the returned
// environment without mutating the original. It deep copies the internal map
// and makes a copy of the Stream struct.
func (m *TestEnv) Clone() *TestEnv {
	if m == nil {
		return nil
	}

	var dataCopy map[string]string
	if m.data != nil {
		dataCopy = make(map[string]string, len(m.data))
		maps.Copy(dataCopy, m.data)
	}

	return &TestEnv{
		jail: m.jail,
		home: m.home,
		user: m.user,
		data: dataCopy,
	}
}

// Ensure implementations satisfy the interfaces.
var _ Env = (*TestEnv)(nil)
