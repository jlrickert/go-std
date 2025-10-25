package testutils

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
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

	// Streamed inputs
	inPipeReader *io.PipeReader
	inPipeWriter *io.PipeWriter

	outBuf *bytes.Buffer
	errBuf *bytes.Buffer
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

	// Create a single pipe for the fixture stdin which can be written to multiple
	// times by tests calling WriteToStdin.
	pr, pw := io.Pipe()

	// Default to not piped; tests may opt-in via WithStdinPiped or WriteToStdin.
	env.SetStdioPiped(false)
	env.SetTTY(false)

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	env.SetStdio(pr)
	env.SetStdout(outBuf)
	env.SetStderr(errBuf)

	lg, handler := std.NewTestLogger(t, std.ParseLevel("debug"))
	clock := std.NewTestClock(time.Date(2025, 10, 15, 12, 30, 0, 0, time.UTC))
	hasher := &std.MD5Hasher{}

	// Populate common temp env vars.
	ctx := t.Context()
	ctx = std.WithLogger(ctx, lg)
	ctx = std.WithEnv(ctx, env)
	ctx = std.WithClock(ctx, clock)
	ctx = std.WithHasher(ctx, hasher)

	f := &Fixture{
		t:            t,
		ctx:          ctx,
		data:         data,
		logger:       handler,
		hasher:       hasher,
		env:          env,
		clock:        clock,
		Jail:         jail,
		inPipeReader: pr,
		inPipeWriter: pw,
		outBuf:       outBuf,
		errBuf:       errBuf,
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

// WithTTY returns a FixtureOption that sets whether stdout should be treated as
// a terminal for the TestEnv. It ensures the underlying Stream is initialized.
func WithTTY(v bool) FixtureOption {
	return func(f *Fixture) {
		f.t.Helper()
		if f.env == nil {
			f.t.Fatalf("WithTTY: fixture Env is nil")
		}
		f.env.SetTTY(v)
	}
}

// WithStdinPiped returns a FixtureOption that sets whether stdin should be
// considered piped/redirected for the TestEnv. It ensures the Stream is
// initialized.
func WithStdinPiped(v bool) FixtureOption {
	return func(f *Fixture) {
		f.t.Helper()
		if f.env == nil {
			f.t.Fatalf("WithStdinPiped: fixture Env is nil")
		}
		f.env.SetStdioPiped(v)
	}
}

// AbsPath returns an absolute path. When the fixture Jail is set and rel is
// relative the path is made relative to the Jail. Otherwise the function
// returns the absolute form of rel.
func (f *Fixture) AbsPath(rel string) string {
	f.t.Helper()
	p, _ := std.ExpandPath(f.Context(), rel)
	return std.EnsureInJailFor(f.Jail, std.AbsPath(f.Context(), p))
}

// Context returns the fixture context.
func (f *Fixture) Context() context.Context {
	return f.ctx
}

// ReadJailFile reads a file located under the fixture Jail. The path is
// interpreted relative to the Jail root.
func (f *Fixture) ReadJailFile(path string) ([]byte, error) {
	f.t.Helper()
	return std.ReadFile(f.Context(), f.AbsPath(path))
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

// ReadStdout returns the current contents written to the fixture stdout
// capture file. It reads the entire file and returns the bytes.
func (f *Fixture) ReadStdout() []byte {
	f.t.Helper()
	return f.outBuf.Bytes()
}

// MustReadStdout reads the fixture stdout capture and fails the test on error.
func (f *Fixture) MustReadStdout() []byte {
	f.t.Helper()
	b := f.ReadStdout()
	if len(b) == 0 {
		f.t.Fatalf("MustReadStdout must read content")
	}
	return b
}

// ReadStderr returns the current contents written to the fixture stderr
func (f *Fixture) ReadStderr() []byte {
	f.t.Helper()
	return f.errBuf.Bytes()
}

// MustReadStdout reads the fixture stdout capture and fails the test on error.
func (f *Fixture) MustReadStderr() []byte {
	f.t.Helper()
	b := f.ReadStderr()
	if len(b) == 0 {
		f.t.Fatalf("MustReadStderr must read content")
	}
	return b
}

// Stdin returns the reader used as the fixture stdin. If the fixture stdin is
// not initialized it falls back to the real process stdin.
func (f *Fixture) Stdin() io.Reader {
	f.t.Helper()
	if f.inPipeReader == nil {
		return os.Stdin
	}
	return f.inPipeReader
}

// Stdout returns the writer capturing fixture stdout. If not initialized it
// falls back to os.Stdout.
func (f *Fixture) Stdout() io.Writer {
	f.t.Helper()
	if f.outBuf == nil {
		return os.Stdout
	}
	return f.outBuf
}

// Stderr returns the writer capturing fixture stderr. If not initialized it
// falls back to os.Stderr.
func (f *Fixture) Stderr() io.Writer {
	f.t.Helper()
	if f.errBuf == nil {
		return os.Stderr
	}
	return f.errBuf
}

// ResolvePath returns a resolved path under the fixture Jail. It does not
// consult the real filesystem; it merely ensures the path is located within
// the Jail when possible.
func (f *Fixture) ResolvePath(path string) string {
	f.t.Helper()
	p, _ := std.ExpandPath(f.Context(), path)
	return std.EnsureInJailFor(f.Jail, p)
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
	// Close the shared stdin writer and restore Stream defaults.
	f.inPipeReader.Close()
	f.inPipeWriter.Close()
	if f.env != nil {
		f.env.SetStdioPiped(false)
		f.env.SetTTY(false)
		f.env.SetStdio(nil)
		f.env.SetStdout(nil)
		f.env.SetStderr(nil)
	}
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

// WriteToStdin writes the provided initial content to the fixture stdin
// writer and returns the writer so tests can write more if desired.
//
// The fixture creates a pipe in NewFixture and reuses the writer for multiple
// calls, so repeated WriteToStdin calls are supported.
func (f *Fixture) WriteToStdin(data []byte) (int, error) {
	f.t.Helper()

	return f.inPipeWriter.Write(data)
}

func (f *Fixture) CloseStdin() error {
	return f.inPipeWriter.Close()
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
