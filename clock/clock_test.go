package clock_test

import (
	"context"
	"testing"
	"time"

	"github.com/jlrickert/go-std/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestClockBasics(t *testing.T) {
	initial := time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC)
	c := clock.NewTestClock(initial)

	// Now should return the seeded time
	assert.Equal(t, initial, c.Now())

	// Advance moves the clock forward
	c.Advance(2 * time.Hour)
	assert.Equal(t, initial.Add(2*time.Hour), c.Now())

	// Set replaces the current time
	newt := time.Date(2021, time.February, 2, 3, 4, 5, 0, time.UTC)
	c.Set(newt)
	assert.Equal(t, newt, c.Now())
}

func TestWithClockAndClockFromContext(t *testing.T) {
	initial := time.Date(2019, time.March, 3, 4, 5, 6, 0, time.UTC)
	tc := clock.NewTestClock(initial)

	ctx := clock.WithClock(context.Background(), tc)
	got := clock.ClockFromContext(ctx)

	// Ensure we can recover the same TestClock from the context
	gotTC, ok := got.(*clock.TestClock)
	require.True(t, ok, "expected ClockFromContext to return *std.TestClock")
	require.Equal(t, tc, gotTC)
	assert.Equal(t, initial, gotTC.Now())
}

func TestClockFromContextDefaultsWhenNilOrMissing(t *testing.T) {
	// Nil context should return the default clock (OsClock) which uses time.Now.
	c := clock.ClockFromContext(t.Context())
	now := time.Now()
	d := c.Now().Sub(now)
	if d < 0 {
		d = -d
	}
	assert.True(t, d < time.Second, "default clock Now() should be close to time.Now()")

	// Storing a nil clock value should result in returning the default clock.
	ctx := clock.WithClock(context.Background(), nil)
	c2 := clock.ClockFromContext(ctx)
	now2 := time.Now()
	d2 := c2.Now().Sub(now2)
	if d2 < 0 {
		d2 = -d2
	}
	assert.True(t, d2 < time.Second, "clock from context with nil value should fall back to default clock")
}
