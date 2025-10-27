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

// TestHarness_Run_NoStdin verifies a simple harness run where the process does
// not read from stdin at all.
func TestHarness_Run_NoStdin(t *testing.T) {
	t.Parallel()

	f := tu.NewFixture(t, nil)

	// Runner that ignores stdin and writes a single line to stdout.
	runner := func(ctx context.Context, s std.Stream, _ []string) error {
		_, _ = fmt.Fprintln(s.Out, "hello, world")
		return nil
	}

	h := tu.NewHarnessFromFixture(t, f, runner)
	out := h.CaptureStdout()

	err := h.Run(f.Context(), nil)
	require.NoError(t, err)

	assert.Equal(t, "hello, world\n", out.String())
}

// TestHarness_Pipe_ProducerToConsumer demonstrates wiring two harnesses such
// that the stdout of the producer is piped into the stdin of the consumer.
func TestHarness_Pipe_ProducerToConsumer(t *testing.T) {
	t.Parallel()

	f := tu.NewFixture(t, nil)

	// Producer emits a few lines to stdout and then exits.
	producer := func(ctx context.Context, s std.Stream, _ []string) error {
		lines := []string{"alpha", "beta", "gamma"}
		for _, l := range lines {
			_, _ = fmt.Fprintln(s.Out, l)
			// Small pause to exercise concurrent piping behavior.
			time.Sleep(5 * time.Millisecond)
		}
		return nil
	}

	// Consumer reads lines from stdin, uppercases them and writes to stdout.
	consumer := func(ctx context.Context, s std.Stream, _ []string) error {
		sc := bufio.NewScanner(s.In)
		for sc.Scan() {
			line := sc.Text()
			_, _ = fmt.Fprintln(s.Out, "C:"+strings.ToUpper(line))
		}
		return sc.Err()
	}

	hProd := tu.NewHarnessFromFixture(t, f, producer)
	hCons := tu.NewHarnessFromFixture(t, f, consumer)

	// Wire producer stdout to consumer stdin.
	r := hProd.StdoutPipe()
	hCons.SetStdinFromReader(r)

	outBuf := hCons.CaptureStdout()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Go(func() {
		errCh <- hProd.Run(f.Context(), nil)
	})

	wg.Go(func() {
		errCh <- hCons.Run(f.Context(), nil)
	})

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	expected := "C:ALPHA\nC:BETA\nC:GAMMA\n"
	assert.Equal(t, expected, outBuf.String())
}
