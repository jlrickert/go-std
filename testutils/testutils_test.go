package testutils_test

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	tu "github.com/jlrickert/go-std/testutils"
	"github.com/stretchr/testify/require"
)

// processStream reads lines from r and writes each line to out and errw.
// It returns when r is closed and the input stream is exhausted.
func processStream(r io.Reader, out io.Writer, errw io.Writer) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		// Echo the same line to both outputs.
		_, _ = fmt.Fprintln(out, line)
		_, _ = fmt.Fprintln(errw, line)
	}
	// Ignore scanning errors for this test helper.
}

// TestNewFixture_ContinuousStdio verifies that callers can push data to the
// fixture stdin continuously and readers attached to the fixture will receive
// the data as it is written.
func TestNewFixture_ContinuousStdio(t *testing.T) {
	t.Parallel()

	// Create a basic fixture. Passing nil options uses defaults.
	f := tu.NewFixture(t, nil, tu.WithStdinPiped(true))

	// Start a processor goroutine that consumes stdin and writes to stdout and
	// stderr.
	var procWG sync.WaitGroup
	procWG.Go(func() {
		processStream(f.Stdin(), f.Stdout(), f.Stderr())
	})

	// Start a writer goroutine that emits lines at a steady pace.
	const linesToWrite = 20
	var writerWG sync.WaitGroup
	writerWG.Go(func() {
		defer f.CloseStdin()
		for i := range linesToWrite {
			_, _ = f.WriteToStdin(fmt.Appendf(nil, "line-%d\n", i))
			time.Sleep(5 * time.Millisecond)
		}
	})

	// Wait for writer to finish and for processor to drain the pipe.
	writerWG.Wait()
	procWG.Wait()

	// Read captured outputs.
	gotOut := f.ReadStdout()
	gotErr := f.ReadStderr()

	// Split into lines, trimming any trailing newline.
	outStr := strings.TrimSpace(string(gotOut))
	errStr := strings.TrimSpace(string(gotErr))

	outLines := []string{}
	if outStr != "" {
		outLines = strings.Split(outStr, "\n")
	}

	errLines := []string{}
	if errStr != "" {
		errLines = strings.Split(errStr, "\n")
	}

	require.Len(t, outLines, linesToWrite)
	require.Len(t, errLines, linesToWrite)

	for i, g := range outLines {
		require.Equal(t, fmt.Sprintf("line-%d", i), g)
	}
	for i, g := range errLines {
		require.Equal(t, fmt.Sprintf("line-%d", i), g)
	}
}
