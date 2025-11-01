package testutils_test

import (
	"testing"

	std "github.com/jlrickert/go-std/pkg"
	tu "github.com/jlrickert/go-std/testutils"
	"github.com/stretchr/testify/require"
)

// TestFixture_BasicSetup verifies Sandbox creation and context
// propagation.
func TestSandbox_BasicSetup(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	ctx := sandbox.Context()
	require.NotNil(t, ctx)

	env := std.EnvFromContext(ctx)
	require.NotNil(t, env)

	logger := std.LoggerFromContext(ctx)
	require.NotNil(t, logger)

	clock := std.ClockFromContext(ctx)
	require.NotNil(t, clock)
}
