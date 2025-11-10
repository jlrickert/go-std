package sandbox_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	tu "github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcess_Run_NoStdin verifies a simple process run where the
// process does not read from stdin at all.
func TestProcess_Run_NoStdin(t *testing.T) {
	t.Parallel()

	// Runner that ignores stdin and writes a single line to stdout.
	runner := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		_, _ = fmt.Fprintln(s.Out, "hello, world")
		_, _ = fmt.Fprintln(s.Err, "Some error!")
		return 0, nil
	}

	h := tu.NewProcess(runner, false)

	result := h.Run(t.Context())
	require.NoError(t, result.Err)

	assert.Equal(t, "hello, world\n", string(result.Stdout))
	assert.Equal(t, "Some error!\n", string(result.Stderr))
}

// TestProcess_Pipe_ProducerToConsumer demonstrates wiring two
// processes such that the stdout of the producer is piped into the
// stdin of the consumer.
func TestProcess_Pipe_ProducerToConsumer(t *testing.T) {
	t.Parallel()

	// Producer emits a few lines to stdout and then exits.
	producer := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		lines := []string{"alpha", "beta", "gamma"}
		for _, l := range lines {
			_, _ = fmt.Fprintln(s.Out, l)
			// Small pause to exercise concurrent piping behavior.
			time.Sleep(5 * time.Millisecond)
		}
		return 0, nil
	}

	// Consumer reads lines from stdin, uppercases them and writes to
	// stdout.
	consumer := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		sc := bufio.NewScanner(s.In)
		for sc.Scan() {
			line := sc.Text()
			_, _ = fmt.Fprintln(s.Out, "C:"+strings.ToUpper(line))
		}
		return 0, sc.Err()
	}

	hProd := tu.NewProcess(producer, false)
	hCons := tu.NewProcess(consumer, false)

	// Wire producer stdout to consumer stdin.
	r := hProd.StdoutPipe()
	hCons.SetStdin(r)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Go(func() {
		res := hProd.Run(t.Context())
		errCh <- res.Err
	})

	wg.Go(func() {
		res := hCons.Run(t.Context())
		errCh <- res.Err
	})

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	expected := "C:ALPHA\nC:BETA\nC:GAMMA\n"
	assert.Equal(t, expected, hCons.CaptureStdout().String())
}

// TestProcess_ContinuousStdin verifies a process can receive
// continuous input on stdin written by the test and that the process
// drains the input while it is written.
func TestProcess_ContinuousStdin(t *testing.T) {
	t.Parallel()

	const linesToWrite = 20

	// Consumer echoes uppercased lines to stdout.
	consumer := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		sc := bufio.NewScanner(s.In)
		for sc.Scan() {
			line := sc.Text()
			_, _ = fmt.Fprintln(s.Out, "C:"+strings.ToUpper(line))
		}
		return 0, sc.Err()
	}

	h := tu.NewProcess(consumer, false)
	out := h.CaptureStdout()

	// Start the process run concurrently.
	errCh := make(chan error, 1)
	go func() {
		res := h.Run(t.Context())
		errCh <- res.Err
	}()

	// Continuously write lines to stdin in a separate goroutine.
	go func() {
		for i := range linesToWrite {
			fmt.Fprintf(h, "line-%d\n", i)
			time.Sleep(5 * time.Millisecond)
		}
		_ = h.Close()
	}()

	// Wait for run to complete.
	err := <-errCh
	require.NoError(t, err)

	// Build expected output and compare.
	var b strings.Builder
	for i := range linesToWrite {
		fmt.Fprintf(&b, "C:LINE-%d\n", i)
	}
	assert.Equal(t, b.String(), out.String())
}

// TestProcess_BufferedStdio verifies that a process can handle
// buffered data on stdio without blocking or data loss.
func TestProcess_BufferedStdio(t *testing.T) {
	t.Parallel()

	const linesToWrite = 50

	// Producer writes a large amount of buffered data to stdout.
	producer := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		w := bufio.NewWriter(s.Out)
		for i := range linesToWrite {
			_, _ = fmt.Fprintf(w, "data-%d\n", i)
		}
		_ = w.Flush()
		return 0, nil
	}

	// Consumer reads buffered data from stdin and writes to stdout.
	consumer := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		r := bufio.NewReader(s.In)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return 1, err
			}
			_, _ = fmt.Fprint(s.Out, "C:"+strings.TrimSpace(line)+"\n")
		}
		return 0, nil
	}

	hProd := tu.NewProcess(producer, false)
	hCons := tu.NewProcess(consumer, false)

	// Wire producer stdout to consumer stdin.
	r := hProd.StdoutPipe()
	hCons.SetStdin(r)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Go(func() {
		res := hProd.Run(t.Context())
		errCh <- res.Err
	})

	wg.Go(func() {
		res := hCons.Run(t.Context())
		errCh <- res.Err
	})

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	// Verify all buffered data was processed.
	output := hCons.CaptureStdout().String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, linesToWrite, len(lines),
		"expected %d lines but got %d", linesToWrite, len(lines))

	// Verify data integrity.
	for i := range linesToWrite {
		expected := fmt.Sprintf("C:data-%d", i)
		assert.Equal(t, expected, lines[i])
	}
}

// TestProcess_RunWithIO verifies that RunWithIO correctly sets the
// input stream and executes the process with provided data.
func TestProcess_RunWithIO(t *testing.T) {
	t.Parallel()

	// Consumer reads from stdin and uppercases the output.
	consumer := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		sc := bufio.NewScanner(s.In)
		for sc.Scan() {
			line := sc.Text()
			_, _ = fmt.Fprintln(s.Out, strings.ToUpper(line))
		}
		return 0, sc.Err()
	}

	h := tu.NewProcess(consumer, false)
	out := h.CaptureStdout()

	// Create input data
	inputData := "line one\nline two\nline three\n"
	inputReader := strings.NewReader(inputData)

	// Run with the provided input
	result := h.RunWithIO(t.Context(), inputReader)
	require.NoError(t, result.Err)
	assert.Equal(t, 0, result.ExitCode)

	// Verify the output
	expected := "LINE ONE\nLINE TWO\nLINE THREE\n"
	assert.Equal(t, expected, out.String())
	assert.Equal(t, expected, string(result.Stdout))
}
