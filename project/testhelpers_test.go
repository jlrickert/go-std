package project_test

import (
	"embed"
	"testing"

	"github.com/jlrickert/go-std/testutils"
)

//go:embed data/**
var testdata embed.FS

func NewFixture(t *testing.T, opts ...testutils.SandboxOption) *testutils.Sandbox {
	return testutils.NewSandbox(t,
		&testutils.SandboxOptions{
			Data: testdata,
			Home: "/home/testuser",
			User: "testuser",
		}, opts...)
}
