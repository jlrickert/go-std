package project_test

import (
	"path/filepath"
	"testing"

	std "github.com/jlrickert/go-std/pkg"
	proj "github.com/jlrickert/go-std/project"
	testutils "github.com/jlrickert/go-std/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectManualRootDefaults(t *testing.T) {
	t.Parallel()

	// Create fixture and populate with the example repo under "repo".
	f := NewSandbox(t,
		testutils.WithFixture("basic", "repo"),
		testutils.WithWd("repo"),
	)

	// Build the project manually without using NewProject. Set the root to the
	// requested absolute path.
	appname := "myapp"
	manualRoot := filepath.FromSlash("/home/testuser/repo/basic")
	p := &proj.Project{
		Appname: appname,
		Root:    manualRoot,
	}

	// Root should be exactly what we set.
	assert.Equal(t, manualRoot, p.Root)

	// Verify config/data/state/cache roots align with user-scoped paths joined
	// with the application name.
	ucfg, err := std.UserConfigPath(f.Context())
	require.NoError(t, err)
	expectedCfg := filepath.Join(ucfg, appname)
	cfg, err := p.ConfigRoot(f.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedCfg, cfg)

	udata, err := std.UserDataPath(f.Context())
	require.NoError(t, err)
	expectedData := filepath.Join(udata, appname)
	data, err := p.DataRoot(f.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedData, data)

	ustate, err := std.UserStatePath(f.Context())
	require.NoError(t, err)
	expectedState := filepath.Join(ustate, appname)
	st, err := p.StateRoot(f.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedState, st)

	ucache, err := std.UserCachePath(f.Context())
	require.NoError(t, err)
	expectedCache := filepath.Join(ucache, appname)
	ca, err := p.CacheRoot(f.Context())
	require.NoError(t, err)
	assert.Equal(t, expectedCache, ca)
}
