package std

import (
	"io"
	"os"

	"golang.org/x/term"
)

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

// Deprecated: CreateTestStdio is unstable and may change in future releases.
//
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
