package testutils

import (
	"context"
	"embed"
	"fmt"
	iofs "io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	std "github.com/jlrickert/go-std/pkg"
)

// FixtureOption is a function used to modify a Fixture during construction.
type FixtureOption func(f *Fixture)

// Fixture bundles common test setup used by package tests. It contains a
// testing.T, a context carrying a test logger, a test env, a test clock, a
// hasher, and a temporary "jail" directory that acts as an isolated filesystem.
type Fixture struct {
	t *testing.T

	data embed.FS
	ctx  context.Context

	logger *std.TestHandler
	env    *std.TestEnv
	clock  *std.TestClock
	hasher *std.MD5Hasher

	// Optional runtime state. Jail is a temporary directory that acts as the
	// root filesystem for file-based test fixtures.
	Jail string
}

// FixtureOptions holds optional settings provided to NewFixture.
type FixtureOptions struct {
	Data embed.FS
	// Home is the home for the user. If empty defaults to /home/$USER.
	// If the user is root the default is /.root.
	Home string
	// User is the user. Defaults to testuser.
	User string
}

// NewFixture constructs a Fixture and applies given options. Cleanup is
// registered with t.Cleanup so callers do not need to call a cleanup func.
func NewFixture(t *testing.T, options *FixtureOptions, opts ...FixtureOption) *Fixture {
	jail := t.TempDir()

	var home string
	var user string
	var data embed.FS
	if options != nil {
		home = options.Home
		user = options.User
		data = options.Data
	}
	env := std.NewTestEnv(jail, home, user)
	lg, handler := std.NewTestLogger(t, std.ParseLevel("debug"))
	clock := std.NewTestClock(time.Now())
	hasher := &std.MD5Hasher{}

	// Populate common temp env vars.
	ctx := t.Context()
	ctx = std.WithLogger(ctx, lg)
	ctx = std.WithEnv(ctx, env)
	ctx = std.WithClock(ctx, clock)
	ctx = std.WithHasher(ctx, hasher)

	f := &Fixture{
		t:      t,
		ctx:    ctx,
		data:   data,
		logger: handler,
		hasher: hasher,
		env:    env,
		clock:  clock,
		Jail:   jail,
	}

	// Apply options.
	for _, opt := range opts {
		opt(f)
	}

	// Register cleanup (reserved for future teardown).
	t.Cleanup(func() { f.cleanup() })

	return f
}

// WithEnv sets a single env entry in the fixture's Env.
func WithEnv(key, val string) FixtureOption {
	return func(f *Fixture) {
		f.t.Helper()
		if f.env == nil {
			f.t.Fatalf("WithEnv: fixture Env is nil")
		}
		if err := f.env.Set(key, val); err != nil {
			f.t.Fatalf("WithEnv failed to set %s: %v", key, err)
		}
	}
}

// WithWd returns a FixtureOption that sets the fixture working directory.
func WithWd(path string) FixtureOption {
	return func(f *Fixture) {
		f.t.Helper()
		f.env.Setwd(path)
	}
}

// WithClock sets the test clock to the provided time.
func WithClock(t0 time.Time) FixtureOption {
	return func(f *Fixture) {
		f.t.Helper()
		if f.clock == nil {
			f.t.Fatalf("WithClock: fixture Clock is nil")
		}
		f.clock.Set(t0)
	}
}

// WithEnvMap seeds multiple environment variables from a map.
func WithEnvMap(m map[string]string) FixtureOption {
	return func(f *Fixture) {
		f.t.Helper()
		for k, v := range m {
			if err := f.env.Set(k, v); err != nil {
				f.t.Fatalf("WithEnvMap set %s failed: %v", k, err)
			}
		}
	}
}

// WithFixture copies a fixture directory from the embedded package data into
// the provided path within the fixture Jail. Example fixtures are "empty" or
// "example".
func WithFixture(fixture string, path string) FixtureOption {
	return func(f *Fixture) {
		f.t.Helper()

		// Source is the embedded package data directory.
		src := filepath.Join("data", fixture)
		if _, err := iofs.Stat(f.data, src); err != nil {
			f.t.Fatalf("WithFileKeg: source %s not found: %v", src, err)
		}

		p, _ := std.ExpandPath(f.Context(), path)
		dst := std.EnsureInJailFor(f.Jail, p)
		if err := copyEmbedDir(f.data, src, dst); err != nil {
			f.t.Fatalf("WithFileKeg: copy %s -> %s failed: %v", src, dst, err)
		}
	}
}

// AbsPath returns an absolute path. When the fixture Jail is set and rel is
// relative the path is made relative to the Jail. Otherwise the function
// returns the absolute form of rel.
func (f *Fixture) AbsPath(rel string) string {
	f.t.Helper()
	p, _ := std.ExpandPath(f.Context(), rel)
	if filepath.IsAbs(p) || f.Jail == "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			f.t.Fatalf("AbsPath failed: %v", err)
		}
		return abs
	}
	return std.AbsPath(f.ctx, filepath.Join(f.Jail, p))
}

// Context returns the fixture context.
func (f *Fixture) Context() context.Context {
	return f.ctx
}

// ReadJailFile reads a file located under the fixture Jail. The path is
// interpreted relative to the Jail root.
func (f *Fixture) ReadJailFile(path string) ([]byte, error) {
	f.t.Helper()
	if f.Jail == "" {
		return nil, fmt.Errorf("no jail set")
	}
	path, err := std.ExpandPath(f.Context(), path)
	if err != nil {
		return nil, err
	}
	path = std.EnsureInJail(f.Jail, path)
	return f.env.ReadFile(path)
}

// MustReadJailFile reads a file under the Jail and fails the test on error.
func (f *Fixture) MustReadJailFile(rel string) []byte {
	f.t.Helper()
	b, err := f.ReadJailFile(rel)
	if err != nil {
		f.t.Fatalf("MustReadJailFile %s failed: %v", rel, err)
	}
	return b
}

// ResolvePath returns a resolved path under the fixture Jail. It does not
// consult the real filesystem; it merely ensures the path is located within
// the Jail when possible.
func (f *Fixture) ResolvePath(path string) string {
	p, _ := std.ExpandPath(f.Context(), path)
	return std.EnsureInJail(f.Jail, p)
}

// WriteJailFile writes data to a path under the fixture Jail, creating parent
// directories as needed. perm is applied to the file.
func (f *Fixture) WriteJailFile(path string, data []byte, perm os.FileMode) error {
	f.t.Helper()
	if f.Jail == "" {
		return fmt.Errorf("no jail set")
	}
	p := f.ResolvePath(path)
	return std.AtomicWriteFile(f.Context(), p, data, perm)
}

// MustWriteJailFile writes data under the Jail and fails the test on error.
func (f *Fixture) MustWriteJailFile(path string, data []byte, perm os.FileMode) {
	f.t.Helper()
	if err := f.WriteJailFile(path, data, perm); err != nil {
		f.t.Fatalf("MustWriteJailFile %s failed: %v", path, err)
	}
}

func (f *Fixture) cleanup() {
	// Reserved for future teardown. Stop mocks or remove long-lived artifacts
	// here if needed.
}

// DumpJailTree logs a tree of files rooted at the fixture's Jail. maxDepth
// limits recursion depth. maxDepth <= 0 means unlimited depth.
func (f *Fixture) DumpJailTree(maxDepth int) {
	f.t.Helper()
	if f.Jail == "" {
		f.t.Log("DumpJailTree: no jail set")
		return
	}

	f.t.Logf("Jail tree: %s", f.Jail)
	err := filepath.WalkDir(f.Jail, func(p string, d iofs.DirEntry, err error) error {
		if err != nil {
			f.t.Logf("  error: %v", err)
			return nil
		}
		var path string
		if p == "." {
			path = "/"
		} else {
			path = std.ResolvePath(f.Context(), p)
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
		suffix := ""
		if d.IsDir() {
			suffix = "/"
		}
		f.t.Logf("  %s%s", path, suffix)
		return nil
	})
	if err != nil {
		f.t.Logf("DumpJailTree walk error: %v", err)
	}
}

// Advance advances the fixture test clock by the given duration.
func (f *Fixture) Advance(d time.Duration) {
	f.t.Helper()
	f.clock.Advance(d)
}

// Now returns the current time from the fixture test clock.
func (f *Fixture) Now() time.Time {
	f.t.Helper()
	return f.clock.Now()
}

func (f *Fixture) Getwd() string {
	f.t.Helper()
	wd, _ := f.env.Getwd()
	return wd
}

// copyEmbedDir recursively copies a directory tree from an embedded FS to dst.
func copyEmbedDir(fsys embed.FS, src, dst string) error {
	entries, err := iofs.ReadDir(fsys, src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		s := path.Join(src, e.Name())
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
