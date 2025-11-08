package toolkit_test

import (
	"path/filepath"
	"testing"

	std "github.com/jlrickert/go-std/toolkit"
	"github.com/stretchr/testify/assert"
)

func TestRemoveJailPrefix(t *testing.T) {
	tests := []struct {
		name string
		jail string
		path string
		want string
	}{
		{
			name: "empty jail returns path unchanged",
			jail: "",
			path: "/some/path",
			want: "/some/path",
		},
		{
			name: "path inside jail with single component",
			jail: "/jail",
			path: "/jail/file.txt",
			want: "/file.txt",
		},
		{
			name: "path inside jail with nested components",
			jail: "/jail",
			path: "/jail/dir/subdir/file.txt",
			want: "/dir/subdir/file.txt",
		},
		{
			name: "path equals jail directory",
			jail: "/jail",
			path: "/jail",
			want: "/",
		},
		{
			name: "jail with trailing slash",
			jail: "/jail/",
			path: "/jail/file.txt",
			want: "/file.txt",
		},
		{
			name: "path with trailing slash",
			jail: "/jail",
			path: "/jail/dir/",
			want: "/dir",
		},
		{
			name: "nested jail directories",
			jail: "/jail/home/testuser",
			path: "/jail/home/testuser/docs/file.txt",
			want: "/docs/file.txt",
		},
		{
			name: "jail with similar prefix path outside jail",
			jail: "/jail",
			path: "/jailbreak/file.txt",
			want: "/jailbreak/file.txt",
		},
		{
			name: "path outside jail different root",
			jail: "/jail",
			path: "/other/path/file.txt",
			want: "/other/path/file.txt",
		},
		{
			name: "single character jail",
			jail: "/a",
			path: "/a/b/c",
			want: "/b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := std.RemoveJailPrefix(
				filepath.FromSlash(tt.jail),
				filepath.FromSlash(tt.path))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsInJail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		jail string
		path string
		want bool
	}{
		{
			name: "empty jail allows all paths",
			jail: "",
			path: "/any/path",
			want: true,
		},
		{
			name: "path inside jail",
			jail: "/jail",
			path: "/jail/file.txt",
			want: true,
		},
		{
			name: "path at jail root",
			jail: "/jail",
			path: "/jail",
			want: true,
		},
		{
			name: "nested path inside jail",
			jail: "/jail",
			path: "/jail/home/user/docs/file.txt",
			want: true,
		},
		{
			name: "path outside jail",
			jail: "/jail",
			path: "/other/file.txt",
			want: false,
		},
		{
			name: "path with similar prefix outside jail",
			jail: "/jail",
			path: "/jailbreak/file.txt",
			want: false,
		},
		{
			name: "jail with trailing slash",
			jail: "/jail/",
			path: "/jail/file.txt",
			want: true,
		},
		{
			name: "path with trailing slash",
			jail: "/jail",
			path: "/jail/dir/",
			want: true,
		},
		{
			name: "both with trailing slashes",
			jail: "/jail/",
			path: "/jail/dir/",
			want: true,
		},
		{
			name: "parent directory escape attempt",
			jail: "/jail",
			path: "/jail/../outside/file.txt",
			want: false,
		},
		{
			name: "relative path inside jail",
			jail: "/jail",
			path: "file.txt",
			want: true,
		},
		{
			name: "nested relative path inside jail",
			jail: "/jail",
			path: "dir/subdir/file.txt",
			want: true,
		},
		{
			name: "deep nesting inside jail",
			jail: "/jail/home/testuser",
			path: "/jail/home/testuser/docs/projects/active/src/main.go",
			want: true,
		},
		{
			name: "one level above jail",
			jail: "/jail/home",
			path: "/jail/other/file.txt",
			want: false,
		},
		{
			name: "single character jail",
			jail: "/a",
			path: "/a/b/c",
			want: true,
		},
		{
			name: "single character jail outside",
			jail: "/a",
			path: "/b/c",
			want: false,
		},
		{
			name: "relative jail with relative path",
			jail: "mydir",
			path: "mydir/file.txt",
			want: true,
		},
		{
			name: "relative jail partial match outside",
			jail: "mydir",
			path: "mydirectory/file.txt",
			want: true,
		},
		{
			name: "tricky",
			jail: "/jail",
			path: "/jail/../a",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := std.IsInJail(
				filepath.FromSlash(tt.jail),
				filepath.FromSlash(tt.path))
			assert.Equal(t, tt.want, got)
		})
	}
}
