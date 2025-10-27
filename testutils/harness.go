package testutils

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"testing"

	std "github.com/jlrickert/go-std/pkg"
)

// Runner is a convenience adapter to allow using plain functions as Runners.
type Runner func(ctx context.Context, stream std.Stream, args []string) error

type Harness struct {
	t *testing.T
	f *Fixture

	IsTTY bool

	// runner to execute
	run Runner

	// Streamed inputs
	inPipeReader *io.PipeReader
	inPipeWriter *io.PipeWriter
	ownInReader  bool // true if harness created the inPipeReader and should close it

	outPipeReader *io.PipeReader
	outPipeWriter *io.PipeWriter
	ownOutWriter  bool // true if harness created the outPipeWriter and should close it

	outBuf *bytes.Buffer
	errBuf *bytes.Buffer

	// cloned env used for the run
	env *std.TestEnv

	mu sync.Mutex
}

// NewHarnessFromFixture constructs a Harness bound to a Fixture and a Runner.
func NewHarnessFromFixture(t *testing.T, f *Fixture, fn Runner) *Harness {
	return &Harness{
		t:   t,
		f:   f,
		run: fn,
	}
}

// FromProcess is a helper that accepts a legacy Process func and returns a Harness.
// It adapts the function to a Runner.
func FromProcess(t *testing.T, proc func(ctx context.Context, stream std.Stream) error) *Harness {
	return &Harness{
		t:   t,
		run: Runner(func(ctx context.Context, s std.Stream, _ []string) error { return proc(ctx, s) }),
	}
}

// StdinWriter creates an io.Pipe and returns the writer side. Tests must close
// the returned writer when done to signal EOF to the process. Calling this
// method twice will fail the test.
func (h *Harness) StdinWriter() io.WriteCloser {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.inPipeWriter != nil || h.inPipeReader != nil {
		h.t.Fatalf("StdinWriter: stdin already configured")
	}
	pr, pw := io.Pipe()
	h.inPipeReader = pr
	h.inPipeWriter = pw
	h.ownInReader = true
	return pw
}

// StdoutPipe creates a pipe for the harness stdout and returns the reader.
// Consecutive calls return the same reader. The harness will close the writer
// side when the run completes to signal EOF to downstream consumers.
func (h *Harness) StdoutPipe() io.ReadCloser {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.outPipeReader != nil {
		return h.outPipeReader
	}
	pr, pw := io.Pipe()
	h.outPipeReader = pr
	h.outPipeWriter = pw
	h.ownOutWriter = true
	return pr
}

// SetStdinFromReader sets the harness stdin to read from the provided reader.
// If the provided reader is not an io.ReadCloser it will be wrapped with
// io.NopCloser. Calling this when stdin is already configured fails the test.
func (h *Harness) SetStdinFromReader(r io.Reader) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.inPipeReader != nil {
		h.t.Fatalf("SetStdinFromReader: stdin already configured")
	}
	if rc, ok := r.(io.ReadCloser); ok {
		h.inPipeReader = rc.(*io.PipeReader)
		// It is an external ReadCloser; harness does not own it.
		h.ownInReader = false
		return
	}
	h.inPipeReader = io.NopCloser(r).(*io.PipeReader)
	// When wrapping a non-ReadCloser we do not own the original; but since
	// we forced a conversion to *io.PipeReader above this branch should not
	// normally be hit with non-pipe readers. Keep ownInReader false.
	h.ownInReader = false
}

// CaptureStdout configures capturing of stdout and returns the buffer that
// will receive captured bytes.
func (h *Harness) CaptureStdout() *bytes.Buffer {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.outBuf == nil {
		h.outBuf = &bytes.Buffer{}
	}
	return h.outBuf
}

// CaptureStderr configures capturing of stderr and returns the buffer that
// will receive captured bytes.
func (h *Harness) CaptureStderr() *bytes.Buffer {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.errBuf == nil {
		h.errBuf = &bytes.Buffer{}
	}
	return h.errBuf
}

// In returns the harness stdin reader (may be nil).
func (h *Harness) In() io.Reader {
	return h.inPipeReader
}

// Out returns the harness stdout writer (may be nil).
func (h *Harness) Out() io.Writer {
	if h.outBuf != nil && h.outPipeWriter == nil {
		return h.outBuf
	}
	return h.outPipeWriter
}

// Err returns the harness stderr writer (may be nil).
func (h *Harness) Err() io.Writer {
	return h.errBuf
}

// Run executes the harnessed runner synchronously following the lifecycle
// described in the specification. It clones the fixture TestEnv, wires streams,
// installs them into the cloned env, then invokes the runner. It closes any
// harness-owned writers/readers when the run completes.
func (h *Harness) Run(ctx context.Context, args []string) error {
	h.mu.Lock()
	if h.f == nil {
		h.mu.Unlock()
		h.t.Fatalf("Run: harness not bound to a Fixture")
	}
	// Clone per-run TestEnv from Fixture.
	clonedEnv := h.f.env.Clone()
	h.env = clonedEnv
	h.mu.Unlock()

	// Prepare streams.
	var in io.Reader = os.Stdin
	if h.inPipeReader != nil {
		in = h.inPipeReader
		// indicate piped stdin on env
		clonedEnv.SetStdioPiped(true)
	}

	var outWriter io.Writer = os.Stdout
	if h.outPipeWriter != nil {
		outWriter = h.outPipeWriter
	}
	// If capture requested, compose writers.
	if h.outBuf != nil {
		if h.outPipeWriter != nil {
			outWriter = io.MultiWriter(h.outPipeWriter, h.outBuf)
		} else {
			outWriter = h.outBuf
		}
	}

	var errWriter io.Writer = os.Stderr
	if h.errBuf != nil {
		errWriter = h.errBuf
	}

	// Install streams into the cloned env so std.StreamFromContext works.
	clonedEnv.SetStdio(in)
	clonedEnv.SetStdout(outWriter)
	clonedEnv.SetStderr(errWriter)

	// Build std.Stream for the runner.
	stream := std.Stream{
		In:      in,
		Out:     outWriter,
		Err:     errWriter,
		IsPiped: h.inPipeReader != nil,
		IsTTY:   h.IsTTY,
	}

	// Build proc context that preserves fixture context values but overrides Env.
	procCtx := std.WithEnv(h.f.Context(), clonedEnv)

	// Invoke the runner (synchronously).
	var runErr error
	if h.run == nil {
		h.t.Fatalf("Run: no runner configured")
	}
	runErr = h.run(procCtx, stream, args)

	// Close any harness-owned writers/readers to signal EOF and free resources.
	h.mu.Lock()
	if h.ownOutWriter && h.outPipeWriter != nil {
		_ = h.outPipeWriter.Close()
	}
	if h.ownInReader && h.inPipeReader != nil {
		_ = h.inPipeReader.Close()
	}
	h.mu.Unlock()

	return runErr
}
