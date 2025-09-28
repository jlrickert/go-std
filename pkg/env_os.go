package std

import (
	"os"
	"os/user"
	"runtime"
)

// OsEnv implements Env by delegating to the real process environment.
// Use this in production code.
type OsEnv struct{}

// Ensure implementations satisfy the interfaces.
var _ Env = (*OsEnv)(nil)

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

func (o *OsEnv) Has(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
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
func (o *OsEnv) Getwd() (string, error) {
	return os.Getwd()
}

// SetWd attempts to change the process working directory to dir and also sets
// the PWD environment variable. Errors from chdir are intentionally ignored by
// this method; callers who need to observe chdir errors can perform os.Chdir
// directly.
func (o *OsEnv) Setwd(dir string) {
	_ = os.Setenv("PWD", dir)
	_ = os.Chdir(dir)
}

func (o *OsEnv) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (o *OsEnv) Mkdir(path string, perm os.FileMode, all bool) error {
	if all {
		return os.MkdirAll(path, perm)
	}
	return os.Mkdir(path, perm)
}
