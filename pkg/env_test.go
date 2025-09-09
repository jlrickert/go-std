package std_test

import (
	"testing"

	"github.com/jlrickert/go-std/pkg"
)

func TestGetDefault(t *testing.T) {
	env := std.NewTestEnv("", "")
	if err := env.Set("EXIST", "val"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if got := std.GetDefault(env, "EXIST", "other"); got != "val" {
		t.Fatalf("GetDefault(EXIST) = %q, want %q", got, "val")
	}

	// empty value should fall back to provided default
	if err := env.Set("EMPTY", ""); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if got := std.GetDefault(env, "EMPTY", "def"); got != "def" {
		t.Fatalf("GetDefault(EMPTY) = %q, want %q", got, "def")
	}

	// missing key should return fallback
	if got := std.GetDefault(env, "MISSING", "fallback"); got != "fallback" {
		t.Fatalf("GetDefault(MISSING) = %q, want %q", got, "fallback")
	}
}

func TestMapEnvSetUnsetHomeUser(t *testing.T) {
	m := std.NewTestEnv("/foo/home", "alice")

	home, err := m.GetHome()
	if err != nil {
		t.Fatalf("GetHome returned error: %v", err)
	}
	if home != "/foo/home" {
		t.Fatalf("initial home = %q, want %q", home, "/foo/home")
	}

	if err := m.Set("HOME", "/bar"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	home, err = m.GetHome()
	if err != nil {
		t.Fatalf("GetHome returned error after Set: %v", err)
	}
	if home != "/bar" {
		t.Fatalf("after Set HOME, GetHome = %q, want %q", home, "/bar")
	}

	m.Unset("HOME")
	if _, err := m.GetHome(); err == nil {
		t.Fatalf("expected error after Unset(HOME), got nil")
	}
}
