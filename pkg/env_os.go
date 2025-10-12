package std

import (
	"os"
	"os/user"
	"path/filepath"
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
// On Windows it sets USERPROFILE in addition to HOME to satisfy common callers.
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

// SetUser sets environment values that represent the current user. On
// Windows it sets USERNAME in addition to USER.
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

// Environ returns a copy of the process environment in "key=value" form.
func (o *OsEnv) Environ() []string {
	return os.Environ()
}

// Has reports whether the given environment key is present.
func (o *OsEnv) Has(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// Unset removes the OS environment variable.
func (o *OsEnv) Unset(key string) {
	_ = os.Unsetenv(key)
}

// GetTempDir returns the OS temporary directory.
func (o *OsEnv) GetTempDir() string {
	return os.TempDir()
}

// Getwd returns the current process working directory.
func (o *OsEnv) Getwd() (string, error) {
	return os.Getwd()
}

// Setwd attempts to change the process working directory to dir. It also
// attempts to update PWD. Chdir errors are intentionally ignored here.
func (o *OsEnv) Setwd(dir string) {
	p, _ := filepath.Abs(dir)
	// _ = os.Setenv("PWD", p)
	_ = os.Chdir(p)
}

// ReadFile reads the named file from the real filesystem.
func (o *OsEnv) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFile writes data to a file on the real filesystem with the given
// permissions.
func (o *OsEnv) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// Remove removes the named file.
func (o *OsEnv) Remove(path string, all bool) error {
	if all {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// Rename renames (moves) a file or directory.
func (o *OsEnv) Rename(src, dst string) error {
	return os.Rename(src, dst)
}

// Mkdir creates a directory. If all is true MkdirAll is used.
func (o *OsEnv) Mkdir(path string, perm os.FileMode, all bool) error {
	if all {
		return os.MkdirAll(path, perm)
	}
	return os.Mkdir(path, perm)
}
