package std

import (
	"errors"
	"os"
	"os/user"
)

// Env is a small, descriptive interface for reading and modifying
// environment values. Implementations may read from the real process
// environment (OsEnv) or from an in-memory store useful for tests
// (MapEnv).
type Env interface {
	// Get returns the raw environment value for key (may be empty).
	Get(key string) string

	// Set assigns the environment key to value.
	Set(key, value string)

	// Unset removes the environment key.
	Unset(key string)

	// GetHome returns the user's home directory. Implementations should
	// return an error if the value is not available.
	GetHome() (string, error)

	// GetUser returns the current user's username. Implementations should
	// return an error if the value is not available.
	GetUser() (string, error)
}

// OsEnv implements Env by delegating to the real process environment.
// Use this in production code.
type OsEnv struct{}

// GetHome implements Env by returning the OS-reported user home directory.
func (o *OsEnv) GetHome() (string, error) {
	return os.UserHomeDir()
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

// Get returns the environment variable for key.
func (o *OsEnv) Get(key string) string {
	return os.Getenv(key)
}

// Set implements Env by setting the OS environment variable.
func (o *OsEnv) Set(key string, value string) {
	os.Setenv(key, value)
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

// GetUser implements Env. Returns an error if user is not set.
func (m *MapEnv) GetUser() (string, error) {
	if m == nil || m.user == "" {
		return "", errors.New("user not set in MapEnv")
	}
	return m.user, nil
}

// Set implements Env. If key is "HOME" or "USER" the corresponding field
// is updated, otherwise the value is stored in the internal map.
func (m *MapEnv) Set(key string, value string) {
	if m == nil {
		return
	}
	switch key {
	case "HOME":
		m.home = value
	case "USER":
		m.user = value
	default:
		if m.data == nil {
			m.data = make(map[string]string)
		}
		m.data[key] = value
	}
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
	case "USER":
		m.user = ""
	default:
		if m.data != nil {
			delete(m.data, key)
		}
	}
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

// Ensure implementations satisfy the interfaces.
var _ Env = (*OsEnv)(nil)
var _ Env = (*MapEnv)(nil)
