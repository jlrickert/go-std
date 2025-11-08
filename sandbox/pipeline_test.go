package sandbox_test

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"testing"

	tu "github.com/jlrickert/go-std/sandbox"
	std "github.com/jlrickert/go-std/toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPipeline_EmptyStages verifies that a pipeline with no stages
// fails appropriately.
func TestPipeline_EmptyStages(t *testing.T) {
	t.Parallel()

	pipeline := tu.NewPipeline()

	result := pipeline.Run(t.Context())

	require.Error(t, result.Err)
	assert.Equal(t, 1, result.ExitCode)
}

// TestPipeline_SingleStage demonstrates a pipeline with one stage
// that produces output.
func TestPipeline_SingleStage(t *testing.T) {
	t.Parallel()

	runner := func(ctx context.Context, s *std.Stream) (int, error) {
		_, _ = fmt.Fprintln(s.Out, "single stage output")
		return 0, nil
	}

	pipeline := tu.NewPipeline(
		tu.Stage("producer", runner),
	)

	result := pipeline.Run(t.Context())

	require.NoError(t, result.Err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "single stage output\n", string(result.Stdout))
}

// TestPipeline_TwoStages verifies piping from one stage to another.
func TestPipeline_TwoStages(t *testing.T) {
	t.Parallel()

	producer := func(ctx context.Context, s *std.Stream) (int, error) {
		lines := []string{"alpha", "beta", "gamma"}
		for _, line := range lines {
			_, _ = fmt.Fprintln(s.Out, line)
		}
		return 0, nil
	}

	consumer := func(ctx context.Context, s *std.Stream) (int, error) {
		sc := bufio.NewScanner(s.In)
		for sc.Scan() {
			line := sc.Text()
			_, _ = fmt.Fprintln(s.Out, "C:"+strings.ToUpper(line))
		}
		return 0, sc.Err()
	}

	pipeline := tu.NewPipeline(
		tu.Stage("producer", producer),
		tu.Stage("consumer", consumer),
	)

	outBuf := pipeline.CaptureStdout()
	result := pipeline.Run(t.Context())

	require.NoError(t, result.Err)
	assert.Equal(t, "C:ALPHA\nC:BETA\nC:GAMMA\n", string(result.Stdout))
	assert.Equal(t, outBuf.String(), string(result.Stdout))
}
