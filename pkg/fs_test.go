package std_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	std "github.com/jlrickert/go-std/pkg"
	"github.com/stretchr/testify/assert"
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
				env := std.NewTestEnv("", "/home/testuser", "testuser")
				return std.WithEnv(context.Background(), env)
			},
			input:    "~",
			expected: "/home/testuser",
		},
		{
			name: "tilde with path expands",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/testuser", "testuser")
				return std.WithEnv(context.Background(), env)
			},
			input:    "~/documents/file.txt",
			expected: "/home/testuser/documents/file.txt",
		},
		{
			name: "tilde in middle is not expanded",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/testuser", "testuser")
				return std.WithEnv(context.Background(), env)
			},
			input:    "/tmp/~user/file",
			expected: "/tmp/~user/file",
		},
		{
			name: "relative path converted to absolute",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob")
				return std.WithEnv(context.Background(), env)
			},
			input:    "documents/file.txt",
			expected: "/home/bob/documents/file.txt",
		},
		{
			name: "relative path with dot",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob")
				return std.WithEnv(context.Background(), env)
			},
			input:    "./config",
			expected: "/home/bob/config",
		},
		{
			name: "relative path with dot dot",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/bob", "bob")
				env.Setwd("/home/bob/subdir")
				return std.WithEnv(context.Background(), env)
			},
			input:    "../documents",
			expected: "/home/bob/documents",
		},
		{
			name: "absolute path unchanged",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/bob", "bob")
				return std.WithEnv(context.Background(), env)
			},
			input:    "/etc/passwd",
			expected: "/etc/passwd",
		},
		{
			name: "removes double slashes",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/bob", "bob")
				return std.WithEnv(context.Background(), env)
			},
			input:    "/home//bob//documents",
			expected: "/home/bob/documents",
		},
		{
			name: "removes trailing slash",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/bob", "bob")
				return std.WithEnv(context.Background(), env)
			},
			input:    "/home/bob/",
			expected: "/home/bob",
		},
		{
			name: "handles dot references",
			setup: func(t *testing.T) context.Context {
				env := std.NewTestEnv("", "/home/bob", "bob")
				return std.WithEnv(context.Background(), env)
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
			result := std.AbsPath(ctx, tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureInJail(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("skipping EnsureInJail tests on windows")
	}

	tests := []struct {
		name     string
		jail     string
		path     string
		expected string
	}{
		{
			name:     "relative path placed in jail",
			jail:     "/jail/root",
			path:     "foo/bar",
			expected: "/jail/root/foo/bar",
		},
		{
			name:     "absolute path already inside jail",
			jail:     "/jail/root",
			path:     "/jail/root/sub",
			expected: "/jail/root/jail/root/sub",
		},
		{
			name:     "absolute path outside jail falls back to jail base",
			jail:     "/jail/root",
			path:     "/other/path/file.txt",
			expected: "/jail/root/other/path/file.txt",
		},
		{
			name:     "empty input returns jail",
			jail:     "/jail/root",
			path:     "",
			expected: "/jail/root",
		},
		{
			name:     "empty jail returns original absolute path",
			jail:     "",
			path:     "/some/abs",
			expected: "/some/abs",
		},
		{
			name:     "empty jail returns original relative path",
			jail:     "",
			path:     "rel/path",
			expected: "rel/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			j := filepath.FromSlash(tt.jail)
			p := filepath.FromSlash(tt.path)
			want := filepath.Clean(filepath.FromSlash(tt.expected))

			got := std.EnsureInJail(j, p)
			got = filepath.Clean(got)

			assert.Equal(t, want, got)
		})
	}
}
