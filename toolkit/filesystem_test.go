package toolkit_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/jlrickert/go-std/mylog"
	"github.com/jlrickert/go-std/toolkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbsPath(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("skipping AbsPath tests on windows")
	}

	tests := []struct {
		name     string
		setup    func(*testing.T) context.Context
		input    string
		expected string
	}{
		{
			name: "empty path returns empty string",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			input:    "",
			expected: "",
		},
		{
			name: "tilde alone expands to home",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/testuser", "testuser")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "~",
			expected: "/home/testuser",
		},
		{
			name: "tilde with path expands",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/testuser", "testuser")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "~/documents/file.txt",
			expected: "/home/testuser/documents/file.txt",
		},
		{
			name: "tilde in middle is not expanded",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/testuser", "testuser")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "/tmp/~user/file",
			expected: "/tmp/~user/file",
		},
		{
			name: "relative path converted to absolute",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "documents/file.txt",
			expected: "/home/bob/documents/file.txt",
		},
		{
			name: "relative path with dot",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "./config",
			expected: "/home/bob/config",
		},
		{
			name: "relative path with dot dot",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob/subdir")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "../documents",
			expected: "/home/bob/documents",
		},
		{
			name: "absolute path unchanged",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "/etc/passwd",
			expected: "/etc/passwd",
		},
		{
			name: "removes double slashes",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "/home//bob//documents",
			expected: "/home/bob/documents",
		},
		{
			name: "removes trailing slash",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "/home/bob/",
			expected: "/home/bob",
		},
		{
			name: "handles dot references",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(context.Background(), env)
			},
			input:    "/home/./bob/./documents",
			expected: "/home/bob/documents",
		},
		{
			name: "no env in context uses OsEnv",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.setup(t)
			result := toolkit.AbsPath(ctx, tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolvePath(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("skipping ResolvePath tests on windows")
	}

	tests := []struct {
		name     string
		setup    func(*testing.T, context.Context) context.Context
		input    string
		expected string
	}{
		{
			name: "empty path returns empty string",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/testuser", "testuser")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "",
			expected: "/home/testuser",
		},
		{
			name: "tilde alone expands to home",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/testuser", "testuser")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "~",
			expected: "/home/testuser",
		},
		{
			name: "tilde with path expands",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/testuser", "testuser")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "~/documents/file.txt",
			expected: "/home/testuser/documents/file.txt",
		},
		{
			name: "relative path converted to absolute",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "documents/file.txt",
			expected: "/home/bob/documents/file.txt",
		},
		{
			name: "relative path with dot",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "./config",
			expected: "/home/bob/config",
		},
		{
			name: "relative path with dot dot",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob/subdir")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "../documents",
			expected: "/home/bob/documents",
		},
		{
			name: "absolute path unchanged",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "/opt/homebrew/etc/passwd",
			expected: "/opt/homebrew/etc/passwd",
		},
		{
			name: "removes double slashes",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "/home//bob//documents",
			expected: "/home/bob/documents",
		},
		{
			name: "removes trailing slash",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "/home/bob/",
			expected: "/home/bob",
		},
		{
			name: "handles dot references",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				return toolkit.WithEnv(ctx, env)
			},
			input:    "/home/./bob/./documents",
			expected: "/home/bob/documents",
		},
		{
			name: "no env in context uses OsEnv",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				return ctx
			},
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lg, _ := mylog.NewTestLogger(t, mylog.ParseLevel("debug"))
			ctx := mylog.WithLogger(t.Context(), lg)

			ctx = tt.setup(t, ctx)
			result, err := toolkit.ResolvePath(ctx, tt.input, false)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRelativePath(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("skipping RelativePath tests on windows")
	}

	tests := []struct {
		name     string
		setup    func(*testing.T) context.Context
		basepath string
		path     string
		expected string
	}{
		{
			name: "empty path returns empty string",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/bob",
			path:     "",
			expected: "",
		},
		{
			name: "same path returns dot",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/bob",
			path:     "/home/bob",
			expected: ".",
		},
		{
			name: "sibling directory",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/bob",
			path:     "/home/alice",
			expected: "../alice",
		},
		{
			name: "child directory",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/bob",
			path:     "/home/bob/documents",
			expected: "documents",
		},
		{
			name: "relative path with tilde expansion",
			setup: func(t *testing.T) context.Context {
				env := toolkit.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob")
				return toolkit.WithEnv(context.Background(), env)
			},
			basepath: "~",
			path:     "~/documents/file.txt",
			expected: "documents/file.txt",
		},
		{
			name: "nested child directory",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/bob",
			path:     "/home/bob/documents/work/file.txt",
			expected: "documents/work/file.txt",
		},
		{
			name: "parent directory",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/bob/documents",
			path:     "/home/bob",
			expected: "..",
		},
		{
			name: "unrelated path falls back to absolute",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/bob",
			path:     "/var/log/system.log",
			expected: "../../var/log/system.log",
		},
		{
			name: "removes double slashes in result",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home//bob",
			path:     "/home/bob/documents",
			expected: "documents",
		},
		{
			name: "handles dot references in paths",
			setup: func(t *testing.T) context.Context {
				return context.Background()
			},
			basepath: "/home/./bob",
			path:     "/home/bob/documents",
			expected: "documents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.setup(t)
			result := toolkit.RelativePath(ctx, tt.basepath, tt.path)

			assert.Equal(t, tt.expected, result)
		})
	}
}

// func TestIsInJail(t *testing.T) {
// 	t.Parallel()
//
// 	if runtime.GOOS == "windows" {
// 		t.Skip("skipping IsInJail tests on windows")
// 	}
//
// 	tests := []struct {
// 		name     string
// 		jail     string
// 		path     string
// 		expected bool
// 	}{
// 		{
// 			name:     "path already inside jail",
// 			jail:     "/jail/root",
// 			path:     "/jail/root/sub",
// 			expected: true,
// 		},
// 		{
// 			name:     "path at jail root",
// 			jail:     "/jail/root",
// 			path:     "/jail/root",
// 			expected: true,
// 		},
// 		{
// 			name:     "path outside jail",
// 			jail:     "/jail/root",
// 			path:     "/other/path",
// 			expected: false,
// 		},
// 		{
// 			name:     "path with similar prefix but outside",
// 			jail:     "/jail/root",
// 			path:     "/jail/root-other",
// 			expected: false,
// 		},
// 		{
// 			name:     "relative path inside jail",
// 			jail:     "/jail/root",
// 			path:     "sub/dir",
// 			expected: true,
// 		},
// 		{
// 			name:     "relative path",
// 			jail:     "/jail/root",
// 			path:     "../escape",
// 			expected: false,
// 		},
// 		{
// 			name:     "empty jail accepts all",
// 			jail:     "",
// 			path:     "/any/path",
// 			expected: true,
// 		},
// 		{
// 			name:     "empty path with jail",
// 			jail:     "/jail/root",
// 			path:     "",
// 			expected: true,
// 		},
// 		{
// 			name:     "nested deeply inside jail",
// 			jail:     "/jail",
// 			path:     "/jail/a/b/c/d/e/f",
// 			expected: true,
// 		},
// 		{
// 			name:     "parent of jail",
// 			jail:     "/jail/root",
// 			path:     "/jail",
// 			expected: false,
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			j := filepath.FromSlash(tt.jail)
// 			p := filepath.FromSlash(tt.path)
//
// 			got := std.IsInJail(j, p)
//
// 			assert.Equal(t, tt.expected, got)
// 		})
// 	}
// }
