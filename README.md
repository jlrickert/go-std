# go-std - Helpers for CLI programs and unit tests

Small, focused helpers for command line programs and tests. The library provides
testable abstractions for environment handling, time, file system operations,
logging, hashing, and common user paths. It is designed to make code easier to
test without touching global process state.

## Highlights

- Testable environment via `TestEnv` that does not modify the OS env.
- Test clock via `TestClock` to deterministically control `Now()`.
- File utilities for safe writes, path resolution, and test stdio.
- Small logging helpers built on `log/slog` and a test handler.
- Simple hashing interface with a default MD5 hasher.
- Helpers for user-scoped directories and a `project` helper for app roots.
- `testutils.Fixture` to simplify common test setup.

## Packages

- `pkg` contains the reusable helpers. Notable files:
  - `pkg/env.go`, `pkg/env_os.go`, `pkg/env_testenv.go`, `pkg/env_test.go`
  - `pkg/clock.go`, `pkg/clock_test.go`
  - `pkg/fs.go`, `pkg/fs_test.go`
  - `pkg/logger.go`, `pkg/logger_testutils.go`
  - `pkg/hash.go`
  - `pkg/user.go`, `pkg/user_test.go`
- `project` contains `Project` utilities to locate config, data, state, and
  cache directories for an application.
- `testutils` contains `Fixture` and helpers used by package tests.

## Install

Run the usual go command to add the module to your project:

```
go get github.com/jlrickert/go-std
```

## Example: Testable environment and variable expansion

```
env := std.NewTestEnv("/tmp/jail", "/home/alice", "alice")
_ = env.Set("FOO", "bar")
ctx := std.WithEnv(context.Background(), env)

out := std.ExpandEnv(ctx, "$FOO/baz")
// out == "bar/baz" on unix-like platforms
```

## Example: Test clock

```
tc := std.NewTestClock(time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC))
ctx := std.WithClock(context.Background(), tc)

now := std.ClockFromContext(ctx).Now()
tc.Advance(2 * time.Hour)
// Now reflects advanced time
```

## Example: Atomic file write

```
ctx := std.WithEnv(context.Background(), std.NewTestEnv("", "", ""))
err := std.AtomicWriteFile(ctx, "/tmp/some/file.txt", []byte("data"), 0644)
if err != nil {
    // handle
}
```

## Example: Test logger

```
lg, th := std.NewTestLogger(t, std.ParseLevel("debug"))
ctx := std.WithLogger(context.Background(), lg)
// use ctx in code under test and assert logs in `th`
```

## Example: Project helper

```
p, err := project.NewProject(ctx, "myapp", project.WithRoot("/path/to/repo"))
cfgRoot, _ := p.ConfigRoot(ctx)
// cfgRoot == <user-config-dir>/myapp
```

## Testing

Run all tests with:

```
go test ./...
```

Many helpers provide test-friendly variants and fixtures. See
`testutils.NewFixture` for a common test setup that wires a `TestEnv`,
`TestClock`, test logger, and a temporary jail directory.

## Contributing

Contributions and issues are welcome. Please open an issue or a pull request
with a short description and tests for new behavior.

## Files to inspect

- `pkg/` - core helpers (`env`, `clock`, `fs`, `logger`, `hash`, `user`)
- `project/` - project path helpers
- `testutils/` - test fixtures and helpers
- `project/data/` - example test data used by `project` tests

## Notes

- The package aims to be small and easy to audit. Where possible, tests avoid
  touching the real OS state by using `TestEnv` and `TestClock`.
- See the repository files for detailed behavior and examples.

## License

See the repository root for license information.
