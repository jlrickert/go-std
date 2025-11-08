package sandbox_test

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	std "github.com/jlrickert/go-std/pkg"
	tu "github.com/jlrickert/go-std/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed data/**
var testdata embed.FS

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

// TestFixture_WithFixtures verifies that fixtures can be added to
// a sandbox and are properly copied into the jail.
func TestFixture_WithFixtures(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, &tu.SandboxOptions{
		Data: testdata,
	},
		tu.WithFixture("example", "~/fixtures/example"),
	)

	data := sandbox.MustReadFile("fixtures/example/example.txt")
	require.NotEmpty(t, data)
	assert.NotNil(t, data)
}

// TestSandbox_ReadWriteStaysInJail verifies that file operations
// remain within the sandbox jail boundary.
func TestSandbox_ReadWriteStaysInJail(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)

	// Write a file within the jail
	testFile := "test.txt"
	testData := []byte("sandbox test content")
	sandbox.MustWriteFile(testFile, testData, 0o644)

	// Verify the file exists within the jail
	readData := sandbox.MustReadFile(testFile)
	assert.Equal(t, testData, readData)

	// Verify the file path is within the jail
	absPath := sandbox.AbsPath(testFile)
	jailAbs, _ := filepath.Abs(sandbox.Jail)
	relPath, err := filepath.Rel(jailAbs, absPath)
	assert.NoError(t, err)
	assert.False(t, filepath.IsAbs(relPath) || relPath == "..",
		"file path %s should be within jail %s", absPath, jailAbs)

	// Verify we cannot escape the jail using relative paths
	escapePath := "../outside.txt"
	resolvedPath := sandbox.ResolvePath(escapePath)
	relPath, err = filepath.Rel(jailAbs, resolvedPath)
	assert.NoError(t, err)
	assert.False(t, filepath.IsAbs(relPath) || relPath == "..",
		"escaped path %s should still be within jail %s",
		resolvedPath, jailAbs)

	// Verify absolute paths outside jail are contained
	outsidePath := "/etc/passwd"
	containedPath := sandbox.ResolvePath(outsidePath)
	relPath, err = filepath.Rel(jailAbs, containedPath)
	assert.NoError(t, err)
	assert.False(t, filepath.IsAbs(relPath) || relPath == "..",
		"absolute path %s should be contained in jail %s",
		containedPath, jailAbs)
}

// TestSandbox_JailIsolation verifies that operations in the sandbox
// do not affect the real filesystem.
func TestSandbox_JailIsolation(t *testing.T) {
	t.Parallel()

	sandbox := tu.NewSandbox(t, nil)
	jailPath := sandbox.Jail

	// Create a test file in the jail
	testFile := "isolated.txt"
	testData := []byte("isolated data")
	sandbox.MustWriteFile(testFile, testData, 0o644)

	// Verify it exists in the jail
	_, err := os.Stat(filepath.Join(jailPath, testFile))
	require.NoError(t, err)

	// Verify it does not exist outside the jail
	tempDirParent := filepath.Dir(jailPath)
	outsidePath := filepath.Join(tempDirParent, testFile)
	_, err = os.Stat(outsidePath)
	require.True(t, os.IsNotExist(err),
		"file should not exist outside jail at %s", outsidePath)
}
