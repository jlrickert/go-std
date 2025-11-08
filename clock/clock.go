package clock

import (
	"context"
	"sync"
	"time"
)

// Clock is a small interface abstracting time operations the package needs.
// Use this to inject testable/fake clocks in unit tests rather than relying on
// time.Now directly.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// OsClock is a production Clock that delegates to time.Now.
type OsClock struct{}

// Now returns the current wall-clock time.
func (OsClock) Now() time.Time { return time.Now() }

// TestClock is a simple, mutex-protected, manually-advancable clock useful for
// tests. It allows deterministic control of Now() by setting an initial time
// and advancing it as needed.
type TestClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewTestClock constructs a TestClock seeded to the provided time.
func NewTestClock(initial time.Time) *TestClock {
	return &TestClock{now: initial}
}

// Now returns the current time of the TestClock.
func (c *TestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the TestClock forward by d.
func (c *TestClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// Set sets the TestClock to a specific time.
func (c *TestClock) Set(t time.Time) {
	c.mu.Lock()
	c.now = t
	c.mu.Unlock()
}

var _ Clock = (*OsClock)(nil)
var _ Clock = (*TestClock)(nil)

type clockCtxKey int

var (
	ctxClockKey  clockCtxKey
	defaultClock = &OsClock{}
)

func WithClock(ctx context.Context, clock Clock) context.Context {
	return context.WithValue(ctx, ctxClockKey, clock)
}

func ClockFromContext(ctx context.Context) Clock {
	if v := ctx.Value(ctxClockKey); v != nil {
		if clock, ok := v.(Clock); ok && clock != nil {
			return clock
		}
	}
	return defaultClock
}
