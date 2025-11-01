package testutils_test

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	std "github.com/jlrickert/go-std/pkg"
	tu "github.com/jlrickert/go-std/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcess_Run_NoStdin verifies a simple process run where the
// process does not read from stdin at all.
func TestProcess_Run_NoStdin(t *testing.T) {
	t.Parallel()

	// Runner that ignores stdin and writes a single line to stdout.
	runner := func(ctx context.Context, s std.Stream) (int, error) {
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
	producer := func(ctx context.Context, s std.Stream) (int, error) {
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
	consumer := func(ctx context.Context, s std.Stream) (int, error) {
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
	consumer := func(ctx context.Context, s std.Stream) (int, error) {
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
