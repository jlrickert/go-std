package std

import (
	"context"
	"io"
	"os"
)

// Stream models the standard IO streams and common stream properties.
//
// Struct field tags are included for clarity to external consumers that may
// wish to encode some stream metadata. The actual reader and writer fields
// are not suitable for encoding and therefore are tagged to be ignored.
type Stream struct {
	// In is the input stream, typically os.Stdin.
	In io.Reader
	// Out is the output stream, typically os.Stdout.
	Out io.Writer
	// Err is the error stream, typically os.Stderr.
	Err io.Writer

	// IsPiped indicates whether stdin appears to be piped or redirected.
	IsPiped bool
	// IsTTY indicates whether stdout refers to a terminal.
	IsTTY bool
}

// streamCtxKey is a private context key type for storing Stream values.
type streamCtxKey int

// ctxStreamKey is the context key used to store and retrieve Stream values.
var ctxStreamKey streamCtxKey

// WithStream returns a copy of ctx that carries the provided Stream.
// Use this to inject custom I/O streams for testing or alternative
// stream configurations.
func WithStream(ctx context.Context, s *Stream) context.Context {
	return context.WithValue(ctx, ctxStreamKey, s)
}

// DefaultStream returns a Stream configured with the real process
// standard input, output, and error streams. It detects whether stdin
// is piped and whether stdout is a terminal.
func DefaultStream() *Stream {
	return &Stream{
		In:      os.Stdin,
		Out:     os.Stdout,
		Err:     os.Stderr,
		IsPiped: StdinHasData(os.Stdin),
		IsTTY:   IsInteractiveTerminal(os.Stdout),
	}
}

// StreamFromContext returns the Stream stored in ctx. If ctx is nil
// or does not contain a Stream, DefaultStream() is returned.
func StreamFromContext(ctx context.Context) *Stream {
	if v := ctx.Value(ctxStreamKey); v != nil {
		if s, ok := v.(*Stream); ok && s != nil {
			return s
		}
	}

	return DefaultStream()
}
