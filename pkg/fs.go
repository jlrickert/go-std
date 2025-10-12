package std

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"log/slog"

	"golang.org/x/term"
)

// Package std contains small helpers for command line programs and test
// utilities for working with standard input/output files and file system
// paths in a cross-platform, testable way.
//
// Functions in this package intentionally accept *os.File so callers can pass
// os.Stdin or a test file created with CreateTestStdio.
//
// StdinHasData reports whether the provided file appears to be receiving
// piped or redirected input (for example: `echo hi | myprog` or
// `myprog < file.txt`).
//
// The implementation performs a lightweight metadata check: it returns true
// when the file is not a character device (that is, not a terminal). This is a
// common, portable heuristic used to detect piped input.
//
// Notes and caveats:
//   - The check does not attempt to read from the file. It only inspects the
//     file mode returned by Stat().
//   - For pipes this indicates stdin is coming from a pipe or redirect, but it
//     does not strictly guarantee that bytes are immediately available to
//     read. An open pipe may be empty until the writer writes to it.
//   - If f.Stat() returns an error the function conservatively returns false.
//   - Callers should pass os.Stdin to check the program's standard input, or a
//     *os.File pointing to another stream for testing.
//
// Example:
//
//	if StdinHasData(os.Stdin) {
//	    // read from stdin
//	}
func StdinHasData(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// IsInteractiveTerminal reports whether the provided file is connected to an
// interactive terminal.
//
// This delegates to golang.org/x/term.IsTerminal and returns true when the
// file descriptor refers to a terminal device (TTY). Pass os.Stdin to check the
// program's standard input.
//
// Notes:
//   - The check uses the file descriptor (f.Fd()) and will return false for
//     pipes, redirected files, and other non-terminal descriptors.
//   - It is a non-destructive check and does not change the file position.
func IsInteractiveTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// CreateTestStdio creates a temporary file prefilled with the given content
// and seeks it to the beginning, making it suitable to pass as a stand-in for
// stdin, stdout, or stderr in tests.
//
// It returns the open *os.File and a cleanup function. The cleanup function
// closes the file and removes it from disk. The function panics on any error
// while creating, writing, or seeking the temporary file; this makes test
// setup failures immediately visible.
//
// Example usage in tests:
//
//	f, cleanup := CreateTestStdio("input text")
//	defer cleanup()
//	// pass f where a *os.File is needed
//
// The returned file is created in the OS temporary directory using the
// pattern "test-stdio-*".
func CreateTestStdio(content string) (*os.File, func()) {
	f, err := os.CreateTemp("", "test-stdio-*")
	if err != nil {
		panic(err)
	}

	if _, err := io.WriteString(f, content); err != nil {
		f.Close()
		os.Remove(f.Name())
		panic(err)
	}

	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		os.Remove(f.Name())
		panic(err)
	}

	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}

	return f, cleanup
}

// AtomicWriteFile writes data to path atomically. It performs the following
// steps:
//   - ensures the parent directory exists,
//   - writes to a temp file in the same directory,
//   - renames the temp file to the final path (atomic on POSIX when on the same
//     filesystem),
//
// The perm parameter is the file mode to use for the final file (for example,
// 0644).
//
// On success the function returns nil. On error it attempts to clean up any
// temporary artifacts and returns a descriptive error.
func AtomicWriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	env := EnvFromContext(ctx)
	lg := LoggerFromContext(ctx)

	path, err := ExpandPath(ctx, path)

	dir := filepath.Dir(path)

	// Ensure parent directory exists.
	if err := Mkdir(ctx, dir, 0o755, true); err != nil {
		lg.Log(
			ctx,
			slog.LevelError,
			"atomic write: mkdirall failed",
			slog.String("dir", dir),
			slog.String("path", path),
			slog.Any("error", err),
		)
		return fmt.Errorf("atomic write: mkdirall %q: %w", dir, err)
	}

	// Create temp file in same dir so rename is atomic on same filesystem.
	env.GetTempDir()
	tmpFile, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+".*")
	if err != nil {
		lg.Log(
			ctx,
			slog.LevelError,
			"atomic write: create temp file failed",
			slog.String("dir", dir),
			slog.Any("error", err),
		)
		return fmt.Errorf("atomic write: create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	// Write data.
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		lg.Log(
			ctx,
			slog.LevelError,
			"atomic write: write temp file failed",
			slog.String("tmp", tmpName),
			slog.Any("error", err),
		)
		return fmt.Errorf("atomic write: write temp file %q: %w", tmpName, err)
	}

	// Close file before renaming.
	if err := tmpFile.Close(); err != nil {
		lg.Log(
			ctx,
			slog.LevelError,
			"atomic write: close temp file failed",
			slog.String("tmp", tmpName),
			slog.Any("error", err),
		)
		return fmt.Errorf("atomic write: close temp file %q: %w", tmpName, err)
	}

	// Set final permissions (rename preserves perms on many systems, but ensure).
	if err := os.Chmod(tmpName, perm); err != nil {
		// Not fatal: attempt rename anyway, but record error if rename fails.
		lg.Log(
			ctx,
			slog.LevelDebug,
			"atomic write: chmod failed, continuing",
			slog.String("tmp", tmpName),
			slog.Any("error", err),
		)
	}

	// Rename into place (atomic on POSIX when same fs).
	if err := Rename(ctx, tmpName, path); err != nil {
		lg.Log(
			ctx,
			slog.LevelError,
			"atomic write: rename failed",
			slog.String("tmp", tmpName),
			slog.String("path", path),
			slog.Any("error", err),
		)
		return fmt.Errorf("atomic write: rename %q -> %q: %w", tmpName, path, err)
	}

	lg.Log(
		ctx,
		slog.LevelDebug,
		"atomic write success",
		slog.String("path", path),
	)
	return nil
}

// AbsPath returns a cleaned absolute path for the provided path. Behavior:
// - If path is empty, returns empty string.
// - Expands a leading tilde using ExpandPath with the Env from ctx.
// - Expands environment variables from the injected env in ctx.
// - If the path is not absolute, attempts to convert it to an absolute path.
// - Returns a cleaned path in all cases.
//
// If ExpandPath fails (for example when HOME is not available) AbsPath falls
// back to the original input and proceeds with expansion of environment
// variables and cleaning.
func AbsPath(ctx context.Context, path string) string {
	if path == "" {
		return ""
	}

	// Expand leading tilde, if present.
	p, err := ExpandPath(ctx, path)
	if err != nil {
		// If expansion fails (for example: no HOME), fall back to the original
		// input.
		p = path
	}

	// If the path is already absolute, just clean and return it.
	if filepath.IsAbs(p) {
		res := filepath.Clean(p)
		return res
	}

	// If a PWD is provided by the injected Env, use it as the base for relative
	// paths. Otherwise fall back to filepath.Abs which uses the process working
	// directory.
	env := EnvFromContext(ctx)
	if cwd, err := env.Getwd(); err == nil && cwd != "" {
		// Ensure cwd is absolute.
		if !filepath.IsAbs(cwd) {
			if absCwd, err := filepath.Abs(cwd); err == nil {
				cwd = absCwd
			}
		}
		joined := filepath.Join(cwd, p)
		res := filepath.Clean(joined)
		return res
	}

	// Fall back to using the process working directory.
	if abs, err := filepath.Abs(p); err == nil {
		res := filepath.Clean(abs)
		return res
	}

	// As a last resort, return the cleaned original.
	res := filepath.Clean(p)
	return res
}

// ResolvePath returns the absolute path with symlinks evaluated. If symlink
// evaluation fails the absolute path returned by AbsPath is returned instead.
func ResolvePath(ctx context.Context, path string) string {
	if path == "" {
		return ""
	}

	abs := AbsPath(ctx, path)
	// Attempt to resolve symlinks; return abs if resolution fails.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		res := filepath.Clean(resolved)
		return res
	}
	return abs
}

// RelativePath returns a path relative to basepath. If path is empty an empty
// string is returned. If computing the relative path fails the absolute
// target path is returned.
func RelativePath(ctx context.Context, basepath, path string) string {
	base := AbsPath(ctx, basepath)
	target := AbsPath(ctx, path)

	rel, err := filepath.Rel(base, target)
	if err != nil {
		// If we cannot compute a relative path, return the absolute target.
		return target
	}
	return rel
}

// findGitRoot attempts to use the git CLI to determine the repository top-level
// directory starting from 'start'. If that fails (git not available, not a git
// worktree, or command error), it falls back to the original upward filesystem
// search for a .git entry.
func FindGitRoot(ctx context.Context, start string) string {
	lg := LoggerFromContext(ctx)

	// Normalize start to a directory (in case a file path was passed).
	if fi, err := Stat(ctx, start); err == nil && !fi.IsDir() {
		start = filepath.Dir(start)
	}

	// First, try using git itself to find the top-level directory. Using `-C`
	// makes git operate relative to the provided path.
	args := []string{"-C", start, "rev-parse", "--show-toplevel"}
	if out, err := exec.CommandContext(ctx, "git", args...).Output(); err == nil {
		if p := strings.TrimSpace(string(out)); p != "" {
			lg.Log(
				ctx,
				slog.LevelDebug,
				"git rev-parse succeeded",
				slog.String("root", p),
			)
			return p
		}
		lg.Log(ctx, slog.LevelDebug, "git rev-parse returned empty output")
	} else {
		lg.Log(
			ctx,
			slog.LevelWarn,
			"git rev-parse failed, falling back",
			slog.String("start", start),
			slog.Any("error", err),
		)
	}

	// Fallback: walk upwards looking for a .git entry (dir or file).
	p := start
	for {
		gitPath := filepath.Join(p, ".git")
		if fi, err := Stat(ctx, gitPath); err == nil {
			// .git can be a dir (normal repo) or a file (worktree / submodule).
			if fi.IsDir() || fi.Mode().IsRegular() {
				lg.Log(ctx, slog.LevelDebug, "found .git entry", slog.String("root", p))
				return p
			}
		}
		parent := filepath.Dir(p)
		if parent == p {
			// reached filesystem root
			break
		}
		p = parent
	}
	lg.Log(ctx, slog.LevelDebug, "git root not found", slog.String("start", start))
	return ""
}

// EnsureInJail returns a path that resides inside jail when possible.
//
// If p is already inside jail the cleaned absolute form of p is returned.
// Otherwise a path under jail is returned by appending the base name of p.
func EnsureInJail(jail, p string) string {
	// Use default logger for non-ctx API.
	if jail == "" {
		return p
	}
	// Clean inputs.
	j := filepath.Clean(jail)
	if p == "" {
		return j
	}
	pp := filepath.Clean(p)

	// If pp is relative, make it absolute relative to jail and return it.
	if !filepath.IsAbs(pp) {
		res := filepath.Join(j, pp)
		return res
	}

	// If pp is within j, return pp as-is.
	rel, err := filepath.Rel(j, pp)
	if err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
		return pp
	}
	// Otherwise, place a safe fallback under jail using the base name.
	res := filepath.Join(jail, pp)
	return res
}

// EnsureInJailFor is a test-friendly helper that mirrors EnsureInJail but
// accepts paths written with forward slashes. It converts both jail and p
// using filepath.FromSlash before applying the EnsureInJail logic.
//
// Use this from tests when expected values are easier to express using
// posix-style literals.
func EnsureInJailFor(jail, p string) string {
	// Convert slash-separated test inputs to platform-specific form.
	j := filepath.FromSlash(jail)
	pp := filepath.FromSlash(p)

	// Reuse EnsureInJail logic on the converted inputs. This ensures tests
	// see the same behavior as production code while allowing easier literals.
	return EnsureInJail(j, pp)
}

// ReadFile reads the named file using the Env stored in ctx. This ensures the
// filesystem view can be controlled by an injected TestEnv.
func ReadFile(ctx context.Context, name string) ([]byte, error) {
	lg := LoggerFromContext(ctx)

	path, err := ExpandPath(ctx, name)
	if err != nil {
		return nil, err
	}
	lg.Log(ctx, slog.LevelDebug, "read file", slog.String("name", name), slog.String("path", path))
	b, err := os.ReadFile(path)
	if err != nil {
		lg.Log(
			ctx,
			slog.LevelError,
			"read file failed",
			slog.String("name", name),
			slog.Any("error", err),
		)
	}
	return b, err
}

// WriteFile writes data to a file using the Env stored in ctx. The perm
// argument is the file mode to apply to the written file.
func WriteFile(ctx context.Context, name string, data []byte, perm os.FileMode) error {
	lg := LoggerFromContext(ctx)
	path, err := ExpandPath(ctx, name)
	if err != nil {
		return err
	}

	lg.Log(
		ctx,
		slog.LevelDebug,
		"write file",
		slog.String("name", name),
		slog.String("path", path),
	)
	if err := os.WriteFile(path, data, perm); err != nil {
		lg.Log(
			ctx,
			slog.LevelError,
			"write file failed",
			slog.String("name", name),
			slog.String("path", path),
			slog.Any("error", err),
		)
		return err
	}
	return nil
}

// Mkdir creates a directory using the Env stored in ctx. If all is true
// MkdirAll is used.
func Mkdir(ctx context.Context, path string, perm os.FileMode, all bool) error {
	lg := LoggerFromContext(ctx)
	lg.Log(
		ctx,
		slog.LevelDebug,
		"mkdir",
		slog.String("path", path),
		slog.Bool("all", all),
	)

	path, err := ExpandPath(ctx, path)
	if err != nil {
		return err
	}

	if all {
		if err := os.MkdirAll(path, perm); err != nil {
			lg.Log(
				ctx,
				slog.LevelError,
				"mkdirAll failed",
				slog.String("path", path),
				slog.Any("error", err),
			)
			return err
		}
	} else {
		if err := os.Mkdir(path, perm); err != nil {
			lg.Log(
				ctx,
				slog.LevelError,
				"mkdir failed",
				slog.String("path", path),
				slog.Any("error", err),
			)
			return err
		}
	}

	return nil
}

// Remove removes the named file or directory using the Env stored in ctx. If
// all is true RemoveAll is used.
func Remove(ctx context.Context, name string, all bool) error {
	lg := LoggerFromContext(ctx)

	path, err := ExpandPath(ctx, name)
	if err != nil {
		return err
	}

	lg.Log(
		ctx,
		slog.LevelDebug,
		"remove",
		slog.String("path", path),
		slog.Bool("all", all),
	)

	if all {
		if err := os.RemoveAll(path); err != nil {
			lg.Log(
				ctx,
				slog.LevelError,
				"removeAll failed",
				slog.String("path", path),
				slog.Any("error", err),
			)
			return err
		}
	} else {
		if err := os.Remove(path); err != nil {
			lg.Log(
				ctx,
				slog.LevelError,
				"remove failed",
				slog.String("path", path),
				slog.Any("error", err),
			)
			return err
		}
	}

	return nil
}

// Rename renames (moves) a file or directory using the Env stored in ctx.
func Rename(ctx context.Context, src, dst string) error {
	lg := LoggerFromContext(ctx)
	lg.Log(
		ctx,
		slog.LevelDebug,
		"rename",
		slog.String("src", src),
		slog.String("dst", dst),
	)

	srcPath, err := ExpandPath(ctx, src)
	if err != nil {
		return err
	}
	dstPath, err := ExpandPath(ctx, dst)
	if err != nil {
		return err
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		lg.Log(
			ctx,
			slog.LevelError,
			"rename failed",
			slog.String("src", src),
			slog.String("dst", dst),
			slog.String("srcPath", srcPath),
			slog.String("dstPath", dstPath),
			slog.Any("error", err),
		)
		return err
	}
	return nil
}

// Stat returns the os.FileInfo for the named file. The path is expanded using
// ExpandPath with the Env from ctx before calling os.Stat.
func Stat(ctx context.Context, name string) (os.FileInfo, error) {
	path, err := ExpandPath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Stat(path)
}

// ReadDir reads the directory named by name and returns a list of entries. The
// path is expanded using ExpandPath with the Env from ctx before calling
// os.ReadDir.
func ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	path, err := ExpandPath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(path)
}
