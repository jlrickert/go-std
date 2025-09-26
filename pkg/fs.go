package std

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// Package std contains small helpers for command line programs and test
// utilities for working with standard input/output files and file system
// paths in a cross-platform, testable way.
//
// Functions in this package intentionally accept *os.File so callers can pass
// os.Stdin or a test file created with CreateTestStdio.

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
//     does not strictly guarantee that bytes are immediately available to read.
//     An open pipe may be empty until the writer writes to it.
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
// - ensures the parent directory exists,
// - writes to a temp file in the same directory,
// - renames the temp file to the final path (atomic on POSIX when on the same filesystem),
//
// The perm parameter is the file mode to use for the final file (for example, 0644).
//
// On success the function returns nil. On error it attempts to clean up any
// temporary artifacts and returns a descriptive error.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Ensure parent directory exists.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("atomic write: mkdirall %q: %w", dir, err)
	}

	// Create temp file in same dir so rename is atomic on same filesystem.
	tmpFile, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("atomic write: create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	// If anything goes wrong, try to remove the temp file.
	cleanup := func() {
		_ = os.Remove(tmpName)
	}

	// Write data.
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return fmt.Errorf("atomic write: write temp file %q: %w", tmpName, err)
	}

	// Close file before renaming.
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("atomic write: close temp file %q: %w", tmpName, err)
	}

	// Set final permissions (rename preserves perms on many systems, but ensure).
	if err := os.Chmod(tmpName, perm); err != nil {
		// Not fatal: attempt rename anyway, but record error if rename fails.
		// We do not return here because chmod may fail on platforms with different semantics.
	}

	// Rename into place (atomic on POSIX when same fs).
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("atomic write: rename %q -> %q: %w", tmpName, path, err)
	}

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
		// If expansion fails (for example: no HOME), fall back to the original input.
		p = path
	}

	// If the path is already absolute, just clean and return it.
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}

	// If a PWD is provided by the injected Env, use it as the base for relative paths.
	// Otherwise fall back to filepath.Abs which uses the process working directory.
	env := EnvFromContext(ctx)
	if cwd, err := env.GetWd(); err == nil && cwd != "" {
		// Ensure cwd is absolute.
		if !filepath.IsAbs(cwd) {
			if absCwd, err := filepath.Abs(cwd); err == nil {
				cwd = absCwd
			}
		}
		joined := filepath.Join(cwd, p)
		return filepath.Clean(joined)
	}

	// Fall back to using the process working directory.
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}

	// As a last resort, return the cleaned original.
	return filepath.Clean(p)
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
		return filepath.Clean(resolved)
	}
	return abs
}

// RelativePath returns a path relative to basepath. If path is empty an empty
// string is returned. If computing the relative path fails the absolute target
// path is returned.
func RelativePath(ctx context.Context, basepath, path string) string {
	if path == "" {
		return ""
	}

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
func FindGitRoot(start string) string {
	// Normalize start to a directory (in case a file path was passed).
	if fi, err := os.Stat(start); err == nil && !fi.IsDir() {
		start = filepath.Dir(start)
	}

	// First, try using git itself to find the top-level directory. Using `-C`
	// makes git operate relative to the provided path.
	if out, err := exec.Command("git", "-C", start,
		"rev-parse", "--show-toplevel").Output(); err == nil {
		if p := strings.TrimSpace(string(out)); p != "" {
			return p
		}
	}

	// Fallback: walk upwards looking for a .git entry (dir or file).
	p := start
	for {
		gitPath := filepath.Join(p, ".git")
		if fi, err := os.Stat(gitPath); err == nil {
			// .git can be a dir (normal repo) or a file (worktree / submodule).
			if fi.IsDir() || fi.Mode().IsRegular() {
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
	return ""
}
