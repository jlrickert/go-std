package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

// Runner is a function signature for executing code within an isolated
// test environment. It receives a context, standard I/O streams, and
// command-line arguments, returning an error on failure.
type Runner func(ctx context.Context, stream *toolkit.Stream) (int, error)

// ProcessResult holds the outcome of process execution including any
// error, exit code, and captured stdout and stderr output.
type ProcessResult struct {
	Err      error
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

// Process manages execution of a Runner function with configurable I/O
// streams and piping support. It allows tests to run functions in
// isolation with piped input/output between multiple processes.
type Process struct {
	args  []string
	isTTY bool

	// runner to execute
	runner Runner

	// I/O streams
	in  io.Reader
	out io.Writer
	err io.Writer

	// Pipes for stdout and stderr
	stdoutPipe *io.PipeReader
	stdoutW    *io.PipeWriter
	stderrPipe *io.PipeReader
	stderrW    *io.PipeWriter

	// Stdin pipe for continuous writing
	stdinPipe *io.PipeReader
	stdinW    *io.PipeWriter

	// Capture buffers
	outBuf *bytes.Buffer
	errBuf *bytes.Buffer

	mu sync.Mutex
}

// NewProcess constructs a Process bound to a Runner function with the
// specified TTY mode. The context parameter is reserved for future use.
func NewProcess(fn Runner, isTTY bool) *Process {
	return &Process{
		runner: fn,
		isTTY:  isTTY,
	}
}

// NewProducer constructs a Process that emits the provided byte buffer
// to stdout. It is useful for testing stages that consume input.
func NewProducer(interval time.Duration, lines []string) *Process {
	runner := func(ctx context.Context, s *toolkit.Stream) (int, error) {
		for _, l := range lines {
			fmt.Fprintln(s.Out, l)
			// Small pause to exercise concurrent piping behavior.
			time.Sleep(interval)
		}
		return 0, nil
	}

	return NewProcess(runner, false)
}

// StdoutPipe returns a reader connected to the process stdout. Writing
// to the process stdout will be readable from the returned reader.
func (p *Process) StdoutPipe() io.Reader {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stdoutPipe == nil {
		p.stdoutPipe, p.stdoutW = io.Pipe()
	}
	return p.stdoutPipe
}

// CaptureStdout configures stdout capture and returns the buffer.
func (p *Process) CaptureStdout() *bytes.Buffer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.outBuf == nil {
		p.outBuf = &bytes.Buffer{}
	}
	return p.outBuf
}

// StderrPipe returns a reader connected to the process stderr. Writing
// to the process stderr will be readable from the returned reader.
func (p *Process) StderrPipe() io.Reader {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stderrPipe == nil {
		p.stderrPipe, p.stderrW = io.Pipe()
	}
	return p.stderrPipe
}

// CaptureStderr configures stderr capture and returns the buffer.
func (p *Process) CaptureStderr() *bytes.Buffer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.errBuf == nil {
		p.errBuf = &bytes.Buffer{}
	}
	return p.errBuf
}

// SetStdin sets the input stream for the process.
func (p *Process) SetStdin(r io.Reader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.in = r
}

// SetStderr sets the error stream for the process.
func (p *Process) SetStderr(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = w
}

// SetStdout sets the output stream for the process.
func (p *Process) SetStdout(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.out = w
}

// SetArgs sets the command-line arguments for the process.
func (p *Process) SetArgs(args []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.args = args
}

// Write writes data to the process stdin. It creates a stdin pipe on
// first call if one does not exist. This allows continuous writing to
// the process while it runs concurrently.
func (p *Process) Write(b []byte) (int, error) {
	p.mu.Lock()
	if p.stdinW == nil {
		p.stdinPipe, p.stdinW = io.Pipe()
		p.in = p.stdinPipe
	}
	w := p.stdinW
	p.mu.Unlock()
	return w.Write(b)
}

// Close closes the process stdin writer. This signals EOF to the
// process, allowing it to complete reading.
func (p *Process) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stdinW != nil {
		return p.stdinW.Close()
	}
	return nil
}

// Run executes the process runner synchronously. It wires configured
// streams, invokes the runner with the provided arguments, and closes
// any process-owned writers and readers when complete. Returns a
// ProcessResult containing the exit code, error, and captured output.
func (p *Process) Run(ctx context.Context) *ProcessResult {
	result := &ProcessResult{}

	// Invoke the runner (synchronously).
	if p.runner == nil {
		result.Err = fmt.Errorf("Run: no runner configured")
		result.ExitCode = 1
		return result
	}

	p.mu.Lock()

	// Setup stdin
	in := p.in
	if in == nil {
		in = bytes.NewReader(nil)
	}

	// Setup stdout
	out := p.out
	if out == nil {
		if p.outBuf != nil {
			out = p.outBuf
		} else if p.stdoutW != nil {
			out = p.stdoutW
		} else {
			out = &bytes.Buffer{}
			p.outBuf = out.(*bytes.Buffer)
		}
	}

	// Setup stderr
	errOut := p.err
	if errOut == nil {
		if p.errBuf != nil {
			errOut = p.errBuf
		} else if p.stderrW != nil {
			errOut = p.stderrW
		} else {
			errOut = &bytes.Buffer{}
			p.errBuf = errOut.(*bytes.Buffer)
		}
	}

	p.mu.Unlock()

	// Build the stream
	stream := &toolkit.Stream{
		In:      in,
		Out:     out,
		Err:     errOut,
		IsPiped: in != nil,
		IsTTY:   p.isTTY,
	}

	// Execute the runner
	exitCode, err := p.runner(ctx, stream)

	// Close pipe writers if they exist
	p.mu.Lock()
	if p.stdoutW != nil {
		p.stdoutW.Close()
	}
	if p.stderrW != nil {
		p.stderrW.Close()
	}
	if p.stdinW != nil {
		p.stdinW.Close()
	}
	p.mu.Unlock()

	// Capture results
	result.Err = err
	result.ExitCode = exitCode

	p.mu.Lock()
	if p.outBuf != nil {
		result.Stdout = p.outBuf.Bytes()
	}
	if p.errBuf != nil {
		result.Stderr = p.errBuf.Bytes()
	}
	p.mu.Unlock()

	return result
}
