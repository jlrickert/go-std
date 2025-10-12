package std_test

import (
	"path/filepath"
	"runtime"
	"testing"

	std "github.com/jlrickert/go-std/pkg"
	"github.com/stretchr/testify/assert"
)

func TestEnsureInJail(t *testing.T) {
	t.Parallel()

	// Do not run on Windows for now.
	if runtime.GOOS == "windows" {
		t.Skip("skipping EnsureInJail tests on windows")
	}

	cases := []struct {
		name     string
		jail     string // written using forward slashes; converted with FromSlash
		p        string // input path
		expected string
	}{
		{
			name:     "relative path placed in jail",
			jail:     "/jail/root",
			p:        "foo/bar",
			expected: "/jail/root/foo/bar",
		},
		{
			name:     "absolute path already inside jail",
			jail:     "/jail/root",
			p:        "/jail/root/sub",
			expected: "/jail/root/sub",
		},
		{
			name:     "absolute path outside jail falls back to jail base",
			jail:     "/jail/root",
			p:        "/other/path/file.txt",
			expected: "/jail/root/other/path/file.txt",
		},
		{
			name:     "empty input returns jail",
			jail:     "/jail/root",
			p:        "",
			expected: "/jail/root",
		},
		{
			name:     "empty jail returns original absolute path",
			jail:     "",
			p:        "/some/abs",
			expected: "/some/abs",
		},
		{
			name:     "empty jail returns original relative path",
			jail:     "",
			p:        "rel/path",
			expected: "rel/path",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			j := filepath.FromSlash(c.jail)
			p := filepath.FromSlash(c.p)
			want := filepath.Clean(filepath.FromSlash(c.expected))

			got := std.EnsureInJail(j, p)
			got = filepath.Clean(got)

			assert.Equal(t, want, got)
		})
	}
}
