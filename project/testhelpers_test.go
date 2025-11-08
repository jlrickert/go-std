package project_test

import (
	"embed"
	"testing"

	testutils "github.com/jlrickert/go-std/sandbox"
)

//go:embed data/**
var testdata embed.FS

func NewSandbox(t *testing.T, opts ...testutils.SandboxOption) *testutils.Sandbox {
	return testutils.NewSandbox(t,
		&testutils.SandboxOptions{
			Data: testdata,
			Home: "/home/testuser",
			User: "testuser",
		}, opts...)
}
