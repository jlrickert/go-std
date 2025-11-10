package sandbox_test

import (
	"path/filepath"
	"testing"

	"github.com/jlrickert/cli-toolkit/clock"
	"github.com/jlrickert/cli-toolkit/mylog"
	tu "github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/stretchr/testify/require"
)

// TestSandbox_BasicSetup verifies Sandbox creation and context
// propagation.
func TestSandbox_BasicSetup(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	ctx := sandbox.Context()
	require.NotNil(t, ctx)

	env := toolkit.EnvFromContext(ctx)
	require.NotNil(t, env)

	logger := mylog.LoggerFromContext(ctx)
	require.NotNil(t, logger)

	clk := clock.ClockFromContext(ctx)
	require.NotNil(t, clk)
}

// TestSandbox_WithFixture ensures fixtures are able to be loaded from
// embedded data.
func TestSandbox_WithFixture(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, &tu.SandboxOptions{
		Data: testdata,
	}, tu.WithFixture("example", "~/fixtures/example"))

	sandbox.DumpJailTree(0)

	data := sandbox.MustReadFile("fixtures/example/example.txt")
	require.NotEmpty(t, data)
}

// TestSandbox_ContextCarriesStream verifies that the stream is
// properly injected into the context.
func TestSandbox_ContextCarriesStream(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	ctx := sandbox.Context()
	stream := toolkit.StreamFromContext(ctx)
	require.NotNil(t, stream)
	require.NotNil(t, stream.In)
	require.NotNil(t, stream.Out)
	require.NotNil(t, stream.Err)
}

// TestSandbox_ContextCarriesHasher verifies that the hasher is
// properly injected into the context.
func TestSandbox_ContextCarriesHasher(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	ctx := sandbox.Context()
	hasher := toolkit.HasherFromContext(ctx)
	require.NotNil(t, hasher)
	require.NotEmpty(t, hasher.Hash([]byte("test")))
}

// TestSandbox_ContextCarriesClock verifies that the test clock is
// available from the context.
func TestSandbox_ContextCarriesClock(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	ctx := sandbox.Context()
	clock := clock.ClockFromContext(ctx)
	require.NotNil(t, clock)

	now := clock.Now()
	require.False(t, now.IsZero())
}

// TestSandbox_ContextCarriesEnv verifies that the test environment is
// available from the context.
func TestSandbox_ContextCarriesEnv(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	ctx := sandbox.Context()
	env := toolkit.EnvFromContext(ctx)
	require.NotNil(t, env)

	home, err := env.GetHome()
	require.NoError(t, err)
	require.NotEmpty(t, home)
}

// TestSandbox_ContextCarriesLogger verifies that the test logger is
// available from the context.
func TestSandbox_ContextCarriesLogger(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	ctx := sandbox.Context()
	logger := mylog.LoggerFromContext(ctx)
	require.NotNil(t, logger)
}

// TestSandbox_MultipleContexts verifies that each sandbox has its own
// independent context.
func TestSandbox_MultipleContexts(t *testing.T) {
	t.Parallel()

	sandbox1 := tu.NewSandbox(t, nil, tu.WithEnv("TEST_KEY", "value1"))
	sandbox2 := tu.NewSandbox(t, nil, tu.WithEnv("TEST_KEY", "value2"))

	env1 := toolkit.EnvFromContext(sandbox1.Context())
	env2 := toolkit.EnvFromContext(sandbox2.Context())

	require.Equal(t, "value1", env1.Get("TEST_KEY"))
	require.Equal(t, "value2", env2.Get("TEST_KEY"))
}

// TestSandbox_ContextPersistsAcrossOperations verifies that context
// modifications persist through operations.
func TestSandbox_ContextPersistsAcrossOperations(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)
	ctx := sandbox.Context()

	env := toolkit.EnvFromContext(ctx)
	err := env.Set("PERSIST_KEY", "persist_value")
	require.NoError(t, err)

	env2 := toolkit.EnvFromContext(ctx)
	require.Equal(t, "persist_value", env2.Get("PERSIST_KEY"))
}

// TestSandbox_ContextWithCustomOptions verifies that custom context
// options are applied correctly.
func TestSandbox_ContextWithCustomOptions(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil,
		tu.WithEnv("CUSTOM_VAR", "custom_value"),
		tu.WithEnv("DEBUG", "true"),
	)

	env := toolkit.EnvFromContext(sandbox.Context())
	require.Equal(t, "custom_value", env.Get("CUSTOM_VAR"))
	require.Equal(t, "true", env.Get("DEBUG"))
}

// TestSandbox_ResolvePath verifies that ResolvePath correctly handles
// various path types and keeps paths within the jail boundary.
func TestSandbox_ResolvePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		cwd      string
	}{
		{
			name:     "relative path",
			input:    "test.txt",
			expected: filepath.Join("/", "home", "testuser", "test.txt"),
		},
		{
			name:     "tilde expansion",
			input:    "~/test.txt",
			expected: filepath.Join("/", "home", "testuser", "test.txt"),
		},
		{
			name:     "escape attempt with dot dot",
			input:    "../../../escape.txt",
			expected: filepath.Join("/escape.txt"),
		},
		{
			name:     "respects working directory",
			cwd:      filepath.Join("~", ".config", "app"),
			input:    "../../repos/GitHub.com",
			expected: filepath.Join("/", "home", "testuser", "repos", "GitHub.com"),
		},
		{
			name:     "absolute path",
			input:    "/opt/etc/passwd",
			expected: filepath.Join("/", "opt", "etc", "passwd"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			sandbox := tu.NewSandbox(t, nil)

			if tc.cwd != "" {
				sandbox.Setwd(tc.cwd)
			}
			resolved := sandbox.ResolvePath(tc.input)
			require.NotEmpty(t, resolved)

			require.Equal(t, tc.expected, resolved, "cwd is %s", sandbox.Getwd())
		})
	}
}
