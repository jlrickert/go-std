package sandbox

import (
	"context"
	"embed"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jlrickert/cli-toolkit/clock"
	"github.com/jlrickert/cli-toolkit/mylog"
	"github.com/jlrickert/cli-toolkit/toolkit"
)

// SandboxOption is a function used to modify a Sandbox during construction.
type SandboxOption func(f *Sandbox)

// Sandbox bundles common test setup used by package tests. It contains a
// testing.T, a context carrying a test logger, a test env, a test clock, a
// hasher, and a temporary "jail" directory that acts as an isolated
// filesystem.
type Sandbox struct {
	t *testing.T

	data embed.FS
	ctx  context.Context

	logger *mylog.TestHandler
	env    *toolkit.TestEnv
	clock  *clock.TestClock
	hasher *toolkit.MD5Hasher
}

// SandboxOptions holds optional settings provided to NewSandbox.
type SandboxOptions struct {
	// Data is an embedded filesystem containing test fixtures.
	Data embed.FS
	// Home is the home directory for the user. If empty defaults to
	// /home/$USER. If the user is root the default is /.root.
	Home string
	// User is the username. Defaults to testuser.
	User string
}

// NewSandbox constructs a Sandbox and applies given options. Cleanup is
// registered with t.Cleanup so callers do not need to call a cleanup
// function.
func NewSandbox(t *testing.T, options *SandboxOptions, opts ...SandboxOption) *Sandbox {
	jail := t.TempDir()

	var home string
	var user string
	var data embed.FS
	if options != nil {
		home = options.Home
		user = options.User
		data = options.Data
	}
	env := toolkit.NewTestEnv(jail, home, user)

	lg, handler := mylog.NewTestLogger(t, mylog.ParseLevel("debug"))
	clk := clock.NewTestClock(
		time.Date(2025, 10, 15, 12, 30, 0, 0, time.UTC))
	hasher := &toolkit.MD5Hasher{}

	// Populate common temp env vars.
	ctx := t.Context()
	ctx = mylog.WithLogger(ctx, lg)
	ctx = toolkit.WithEnv(ctx, env)
	ctx = clock.WithClock(ctx, clk)
	ctx = toolkit.WithHasher(ctx, hasher)

	f := &Sandbox{
		t:      t,
		ctx:    ctx,
		data:   data,
		logger: handler,
		hasher: hasher,
		env:    env,
		clock:  clk,
	}

	// Apply options.
	for _, opt := range opts {
		opt(f)
	}

	// Register cleanup (reserved for future teardown).
	t.Cleanup(func() { f.cleanup() })

	return f
}

// WithEnv returns a SandboxOption that sets a single environment variable
// in the sandbox's Env.
func WithEnv(key, val string) SandboxOption {
	return func(f *Sandbox) {
		f.t.Helper()
		if f.env == nil {
			f.t.Fatalf("WithEnv: sandbox Env is nil")
		}
		if err := f.env.Set(key, val); err != nil {
			f.t.Fatalf("WithEnv failed to set %s: %v", key, err)
		}
	}
}

// WithWd returns a SandboxOption that sets the sandbox working directory.
func WithWd(rel string) SandboxOption {
	return func(sandbox *Sandbox) {
		sandbox.t.Helper()
		path := sandbox.ResolvePath(rel)
		sandbox.env.Setwd(path)
	}
}

// WithClock returns a SandboxOption that sets the test clock to the
// provided time.
func WithClock(t0 time.Time) SandboxOption {
	return func(f *Sandbox) {
		f.t.Helper()
		if f.clock == nil {
			f.t.Fatalf("WithClock: sandbox Clock is nil")
		}
		f.clock.Set(t0)
	}
}

// WithEnvMap returns a SandboxOption that seeds multiple environment
// variables from a map.
func WithEnvMap(m map[string]string) SandboxOption {
	return func(f *Sandbox) {
		f.t.Helper()
		for k, v := range m {
			if err := f.env.Set(k, v); err != nil {
				f.t.Fatalf("WithEnvMap set %s failed: %v", k, err)
			}
		}
	}
}

// WithFixture returns a SandboxOption that copies a fixture directory from
// the embedded package data into the provided path within the sandbox Jail.
// Example fixtures are "empty" or "example".
func WithFixture(fixture string, path string) SandboxOption {
	return func(f *Sandbox) {
		f.t.Helper()

		// Source is the embedded package data directory.
		src := filepath.Join("data", fixture)
		if _, err := iofs.Stat(f.data, src); err != nil {
			f.t.Fatalf("WithFixture: source %s not found: %v", src, err)
		}

		p, _ := toolkit.ResolvePath(f.Context(), path, false)
		dst := filepath.Join(f.GetJail(), p)
		if err := copyEmbedDir(f.data, src, dst); err != nil {
			f.t.Fatalf("WithFixture: copy %s -> %s failed: %v",
				src, dst, err)
		}
	}
}

func (sandbox *Sandbox) GetJail() string {
	return sandbox.env.GetJail()
}

// Context returns the sandbox context.
func (sandbox *Sandbox) Context() context.Context {
	return sandbox.ctx
}

// AbsPath returns an absolute path. When the sandbox Jail is set and rel is
// relative the path is made relative to the Jail. Otherwise the function
// returns the absolute form of rel.
func (sandbox *Sandbox) AbsPath(rel string) string {
	sandbox.t.Helper()
	return toolkit.AbsPath(sandbox.Context(), rel)
}

// ReadFile reads a file located under the sandbox Jail. The path is
// interpreted relative to the Jail root.
func (sandbox *Sandbox) ReadFile(rel string) ([]byte, error) {
	sandbox.t.Helper()
	return toolkit.ReadFile(sandbox.Context(), rel)
}

// MustReadFile reads a file under the Jail and fails the test on error.
func (sandbox *Sandbox) MustReadFile(rel string) []byte {
	sandbox.t.Helper()
	b, err := sandbox.ReadFile(rel)
	if err != nil {
		sandbox.t.Fatalf("MustReadJailFile %s failed: %v", rel, err)
	}
	return b
}

func (sandbox *Sandbox) AtomicWriteFile(rel string, data []byte, perm os.FileMode) error {
	sandbox.t.Helper()
	if sandbox.GetJail() == "" {
		return fmt.Errorf("no jail set")
	}
	return toolkit.AtomicWriteFile(sandbox.Context(), rel, data, perm)
}

// WriteFile writes data to a path under the sandbox Jail, creating
// parent directories as needed. perm is applied to the file.
func (sandbox *Sandbox) WriteFile(rel string, data []byte, perm os.FileMode) error {
	sandbox.t.Helper()
	return toolkit.WriteFile(sandbox.Context(), rel, data, perm)
}

// MustWriteFile writes data under the Jail and fails the test on error.
func (sandbox *Sandbox) MustWriteFile(path string, data []byte, perm os.FileMode) {
	sandbox.t.Helper()
	if err := sandbox.WriteFile(path, data, perm); err != nil {
		sandbox.t.Fatalf("MustWriteJailFile %s failed: %v", path, err)
	}
}

func (sandbox *Sandbox) Mkdir(rel string, all bool) error {
	sandbox.t.Helper()
	return sandbox.env.Mkdir(rel, 0o755, all)
}

// ResolvePath returns an absolute path with symlinks resolved, ensuring the
// path remains within the sandbox Jail. It expands the path and evaluates any
// symbolic links before confining it to the jail boundary.
func (sandbox *Sandbox) ResolvePath(rel string) string {
	sandbox.t.Helper()
	// TODO: handle the error
	path, _ := toolkit.ResolvePath(sandbox.Context(), rel, false)
	return path
}

func (sandbox *Sandbox) cleanup() {
}

// DumpJailTree logs a tree of files and directories rooted at the sandbox's
// Jail. Only directories with no children and files are logged. maxDepth
// limits recursion depth; maxDepth <= 0 means unlimited depth.
func (sandbox *Sandbox) DumpJailTree(maxDepth int) {
	sandbox.t.Helper()
	if sandbox.GetJail() == "" {
		sandbox.t.Log("DumpJailTree: no jail set")
		return
	}

	sandbox.t.Logf("Jail tree: %s", sandbox.GetJail())

	// First pass: collect all paths and determine which dirs have children
	type pathInfo struct {
		path  string
		isDir bool
		depth int
	}
	var paths []pathInfo
	hasDirChild := make(map[string]bool)

	err := filepath.WalkDir(sandbox.GetJail(), func(p string, d iofs.DirEntry,
		err error) error {
		if err != nil {
			sandbox.t.Logf("  error: %v", err)
			return nil
		}

		var path string
		if p == "." {
			path = "/"
		} else {
			// TODO: Handle the error
			path, _ = toolkit.ResolvePath(sandbox.Context(), p, false)
		}

		// Apply depth limit when requested.
		if maxDepth > 0 {
			depth := strings.Count(path, string(os.PathSeparator)) + 1
			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			depth := strings.Count(path, string(os.PathSeparator)) + 1
			paths = append(paths, pathInfo{
				path:  path,
				isDir: true,
				depth: depth,
			})
		} else {
			paths = append(paths, pathInfo{
				path:  path,
				isDir: false,
			})
			// Mark parent as having children
			parent := filepath.Dir(path)
			hasDirChild[parent] = true
		}

		return nil
	})

	// Second pass: log only files and leaf directories
	for _, pi := range paths {
		if !pi.isDir {
			// Always log files
			sandbox.t.Logf("  %s", pi.path)
		} else if !hasDirChild[pi.path] {
			// Log directories with no children
			sandbox.t.Logf("  %s/", pi.path)
		}
	}

	if err != nil {
		sandbox.t.Logf("DumpJailTree walk error: %v", err)
	}
}

// Advance advances the sandbox test clock by the given duration.
func (sandbox *Sandbox) Advance(d time.Duration) {
	sandbox.t.Helper()
	sandbox.clock.Advance(d)
}

// Now returns the current time from the sandbox test clock.
func (sandbox *Sandbox) Now() time.Time {
	sandbox.t.Helper()
	return sandbox.clock.Now()
}

// Getwd returns the sandbox working directory.
func (sandbox *Sandbox) Getwd() string {
	sandbox.t.Helper()
	wd, _ := sandbox.env.Getwd()
	return wd
}

// Setwd sets the sandbox working directory.
func (sandbox *Sandbox) Setwd(dir string) {
	sandbox.t.Helper()
	path := sandbox.ResolvePath(dir)
	sandbox.env.Setwd(path)
}

// GetHome returns the sandbox home.
func (sandbox *Sandbox) GetHome() (string, error) {
	sandbox.t.Helper()
	return sandbox.env.GetHome()
}

// copyEmbedDir recursively copies a directory tree from an embedded FS
// to dst.
func copyEmbedDir(fsys embed.FS, src, dst string) error {
	entries, err := iofs.ReadDir(fsys, src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyEmbedDir(fsys, s, d); err != nil {
				return err
			}
			continue
		}
		data, err := fsys.ReadFile(s)
		if err != nil {
			return err
		}
		if err := os.WriteFile(d, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
