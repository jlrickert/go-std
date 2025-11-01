package testutils_test

import (
	"testing"

	std "github.com/jlrickert/go-std/pkg"
	tu "github.com/jlrickert/go-std/testutils"
	"github.com/stretchr/testify/assert"
)

// TestFixture_WithEnvironment verifies environment variable setup
// via options.
func TestFixture_WithEnvironment(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil,
		tu.WithEnv("DEBUG", "true"),
		tu.WithEnv("LOG_LEVEL", "info"),
	)

	env := std.EnvFromContext(sandbox.Context())
	assert.Equal(t, "true", env.Get("DEBUG"))
	assert.Equal(t, "info", env.Get("LOG_LEVEL"))
}

// // TestFixture_WithJail verifies that the jail provides isolated
// // filesystem operations.
// func TestFixture_WithJail(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
// 	require.NotEmpty(t, sandbox.Jail)
//
// 	// Write a file to the jail
// 	data := []byte("test content")
// 	sandbox.MustWriteJailFile("test.txt", data, 0o644)
//
// 	// Read it back
// 	read := sandbox.MustReadJailFile("test.txt")
// 	assert.Equal(t, data, read)
// }
//
// // TestFixture_ClockControl verifies test clock advancement and
// // snapshot.
// func TestFixture_ClockControl(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil,
// 		tu.WithClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
// 	)
//
// 	initial := sandbox.Now()
// 	assert.Equal(t, 2025, initial.Year())
//
// 	sandbox.Advance(24 * time.Hour)
// 	advanced := sandbox.Now()
// 	assert.Equal(t, 2, advanced.Day())
// }
//
// // TestProcess_SimpleRun verifies a basic Process execution.
// func TestProcess_SimpleRun(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
//
// 	runner := func(ctx context.Context, s std.Stream,
// 		_ []string) error {
// 		_, _ = fmt.Fprintln(s.Out, "test output")
// 		return nil
// 	}
//
// 	process := tu.NewProcessFromFixture(t, sandbox, runner)
// 	out := process.CaptureStdout()
//
// 	err := process.Run(sandbox.Context(), nil)
//
// 	require.NoError(t, err)
// 	assert.Equal(t, "test output\n", out.String())
// }
//
// // TestProcess_WithArguments verifies argument passing to runners.
// func TestProcess_WithArguments(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
//
// 	runner := func(ctx context.Context, s std.Stream,
// 		args []string) error {
// 		for _, arg := range args {
// 			_, _ = fmt.Fprintln(s.Out, arg)
// 		}
// 		return nil
// 	}
//
// 	process := tu.NewProcessFromFixture(t, sandbox, runner)
// 	out := process.CaptureStdout()
//
// 	args := []string{"arg1", "arg2", "arg3"}
// 	err := process.Run(sandbox.Context(), args)
//
// 	require.NoError(t, err)
// 	assert.Equal(t, "arg1\narg2\narg3\n", out.String())
// }
//
// // TestProcess_StdoutAndStderr verifies capture of both streams.
// func TestProcess_StdoutAndStderr(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
//
// 	runner := func(ctx context.Context, s std.Stream,
// 		_ []string) error {
// 		_, _ = fmt.Fprintln(s.Out, "stdout message")
// 		_, _ = fmt.Fprintln(s.Err, "stderr message")
// 		return nil
// 	}
//
// 	process := tu.NewProcessFromFixture(t, sandbox, runner)
// 	out := process.CaptureStdout()
// 	err := process.CaptureStderr()
//
// 	runErr := process.Run(sandbox.Context(), nil)
//
// 	require.NoError(t, runErr)
// 	assert.Equal(t, "stdout message\n", out.String())
// 	assert.Equal(t, "stderr message\n", err.String())
// }
//
// // TestProcess_EnvironmentIsolation verifies that each run gets a
// // cloned environment.
// func TestProcess_EnvironmentIsolation(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil,
// 		tu.WithEnv("VAR", "original"),
// 	)
//
// 	runner := func(ctx context.Context, s std.Stream,
// 		_ []string) error {
// 		env := std.EnvFromContext(ctx)
// 		_ = env.Set("VAR", "modified")
// 		_, _ = fmt.Fprintln(s.Out, env.Get("VAR"))
// 		return nil
// 	}
//
// 	process := tu.NewProcessFromFixture(t, sandbox, runner)
// 	out := process.CaptureStdout()
//
// 	err := process.Run(sandbox.Context(), nil)
//
// 	require.NoError(t, err)
// 	assert.Equal(t, "modified\n", out.String())
//
// 	// Verify original environment is unchanged
// 	env := std.EnvFromContext(sandbox.Context())
// 	assert.Equal(t, "original", env.Get("VAR"))
// }
//
// // TestPipeline_TwoProcesses verifies piping stdout from one process
// // to stdin of another.
// func TestPipeline_TwoProcesses(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
//
// 	producer := func(ctx context.Context, s std.Stream,
// 		_ []string) error {
// 		for _, line := range []string{"one", "two", "three"} {
// 			_, _ = fmt.Fprintln(s.Out, line)
// 		}
// 		return nil
// 	}
//
// 	consumer := func(ctx context.Context, s std.Stream,
// 		_ []string) error {
// 		sc := bufio.NewScanner(s.In)
// 		for sc.Scan() {
// 			_, _ = fmt.Fprintf(s.Out, "C:%s\n",
// 				strings.ToUpper(sc.Text()))
// 		}
// 		return sc.Err()
// 	}
//
// 	p1 := tu.NewProcessFromFixture(t, sandbox, producer)
// 	p2 := tu.NewProcessFromFixture(t, sandbox, consumer)
//
// 	r := p1.StdoutPipe()
// 	p2.SetStdinFromReader(r)
//
// 	p2Out := p2.CaptureStdout()
//
// 	// Run both concurrently
// 	errCh := make(chan error, 2)
// 	go func() { errCh <- p1.Run(sandbox.Context(), nil) }()
// 	go func() { errCh <- p2.Run(sandbox.Context(), nil) }()
//
// 	for range 2 {
// 		require.NoError(t, <-errCh)
// 	}
//
// 	expected := "C:ONE\nC:TWO\nC:THREE\n"
// 	assert.Equal(t, expected, p2Out.String())
// }
//
// // TestPipeline_MultiStageTransformation verifies a three-stage
// // pipeline.
// func TestPipeline_MultiStageTransformation(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
//
// 	pipeline := tu.NewPipelineFromSandbox(t, sandbox,
// 		tu.Stage("numbers", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			for i := 1; i <= 3; i++ {
// 				_, _ = fmt.Fprintf(s.Out, "%d\n", i)
// 			}
// 			return nil
// 		}),
// 		tu.Stage("double", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			sc := bufio.NewScanner(s.In)
// 			for sc.Scan() {
// 				_, _ = fmt.Fprintf(s.Out, "%s %s\n",
// 					sc.Text(), sc.Text())
// 			}
// 			return sc.Err()
// 		}),
// 		tu.Stage("count", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			sc := bufio.NewScanner(s.In)
// 			count := 0
// 			for sc.Scan() {
// 				count++
// 				_, _ = fmt.Fprintf(s.Out, "%d: %s\n",
// 					count, sc.Text())
// 			}
// 			return sc.Err()
// 		}),
// 	)
//
// 	out := pipeline.CaptureStdout()
// 	result := pipeline.Run(sandbox.Context(), nil)
//
// 	require.NoError(t, result.Err)
// 	expected := "1: 1 1\n2: 2 2\n3: 3 3\n"
// 	assert.Equal(t, expected, string(result.Stdout))
// 	assert.Equal(t, expected, out.String())
// }
//
// // TestPipeline_WithTimeout verifies timeout handling in pipelines.
// func TestPipeline_WithTimeout(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
//
// 	pipeline := tu.NewPipelineFromSandbox(t, sandbox,
// 		tu.Stage("fast", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			_, _ = fmt.Fprintln(s.Out, "done")
// 			return nil
// 		}),
// 	)
//
// 	result := pipeline.RunWithTimeout(sandbox.Context(),
// 		5*time.Second, nil)
//
// 	require.NoError(t, result.Err)
// 	assert.Equal(t, "done\n", string(result.Stdout))
// }
//
// // TestPipeline_ErrorPropagation verifies that errors from any stage
// // are captured.
// func TestPipeline_ErrorPropagation(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
//
// 	pipeline := tu.NewPipelineFromSandbox(t, sandbox,
// 		tu.Stage("ok", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			_, _ = fmt.Fprintln(s.Out, "success")
// 			return nil
// 		}),
// 		tu.Stage("fail", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			return fmt.Errorf("intentional error")
// 		}),
// 	)
//
// 	result := pipeline.Run(sandbox.Context(), nil)
//
// 	require.Error(t, result.Err)
// 	assert.Equal(t, 1, result.ExitCode)
// 	assert.Contains(t, result.Err.Error(), "intentional error")
// }
//
// // TestPipeline_HighVolume verifies pipeline handles large data volume.
// func TestPipeline_HighVolume(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil)
// 	const count = 500
//
// 	pipeline := tu.NewPipelineFromSandbox(t, sandbox,
// 		tu.Stage("producer", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			for i := range count {
// 				_, _ = fmt.Fprintf(s.Out, "line %d\n", i)
// 			}
// 			return nil
// 		}),
// 		tu.Stage("counter", func(ctx context.Context, s std.Stream,
// 			_ []string) error {
// 			sc := bufio.NewScanner(s.In)
// 			n := 0
// 			for sc.Scan() {
// 				n++
// 			}
// 			_, _ = fmt.Fprintf(s.Out, "total: %d\n", n)
// 			return sc.Err()
// 		}),
// 	)
//
// 	result := pipeline.Run(sandbox.Context(), nil)
//
// 	require.NoError(t, result.Err)
// 	assert.Equal(t, fmt.Sprintf("total: %d\n", count), string(result.Stdout))
// }
//
// // TestIntegration_CompleteWorkflow verifies a realistic workflow combining
// // fixture, process, and pipeline.
// func TestIntegration_CompleteWorkflow(t *testing.T) {
// 	t.Parallel()
//
// 	sandbox := tu.NewSandbox(t, nil,
// 		tu.WithEnv("MODE", "test"),
// 	)
//
// 	// Setup files in jail
// 	sandbox.MustWriteJailFile("input.txt", []byte("alpha\nbeta\ngamma\n"), 0o644)
//
// 	// Create pipeline that reads, transforms, and counts
// 	pipeline := tu.NewPipelineFromSandbox(t, sandbox,
// 		tu.Stage("read", func(ctx context.Context, s std.Stream, _ []string) error {
// 			data, err := std.ReadFile(ctx, sandbox.AbsPath("input.txt"))
// 			if err != nil {
// 				return err
// 			}
// 			_, _ = fmt.Fprint(s.Out, string(data))
// 			return nil
// 		}),
// 		tu.Stage("transform", func(ctx context.Context, s std.Stream, _ []string) error {
// 			sc := bufio.NewScanner(s.In)
// 			for sc.Scan() {
// 				_, _ = fmt.Fprintf(s.Out, "%s\n",
// 					strings.ToUpper(sc.Text()))
// 			}
// 			return sc.Err()
// 		}),
// 		tu.Stage("count", func(ctx context.Context, s std.Stream, _ []string) error {
// 			sc := bufio.NewScanner(s.In)
// 			n := 0
// 			for sc.Scan() {
// 				n++
// 			}
// 			_, _ = fmt.Fprintf(s.Out, "processed: %d\n", n)
// 			return sc.Err()
// 		}),
// 	)
//
// 	result := pipeline.Run(sandbox.Context(), nil)
//
// 	require.NoError(t, result.Err)
// 	assert.Equal(t, "processed: 3\n", string(result.Stdout))
// }
