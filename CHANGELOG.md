## [0.1.1] - 2025-11-09

### 🐛 Bug Fixes

- Make following symlinks not the default

### 📚 Documentation

- Update README with package organization and examples
## [0.1.0] - 2025-11-08

### 🚀 Features

- Add std package with env, clock, fs, and user helpers
- Add logger, atomic file write, and test env helpers
- Add context helpers and extend Env API
- Add ExpandPath helper to expand leading tilde
- Expand Env interface and add path and env utilities
- Add hasher, testutils, Edit helper and bump deps
- Add project package and improve test utilities
- Add Stream to Env and fixture stdio support
- Add single-process test Harness and improve TestEnv
- Add experimental Stream interface for I/O abstraction
- Add filepath jail utilities for sandboxing paths

### 🐛 Bug Fixes

- Make clock context helpers robust and add tests

### 🚜 Refactor

- Use context-based Env in user and env packages
- Reorganize logger and simplify context helpers
- Split env into OsEnv/TestEnv and normalize Env API
- Centralize filesystem helpers and use context path expansion
- Decouple stream handling from Env interface
- Improve filesystem function signatures and logging
- Move clock package to top-level directory
- Relocate logger to mylog package and enhance LoggedEntry
- Reorganize packages and extract filesystem operations
- Add ResolvePath to Env interface and improve jail confinement
- Rename pkg package to toolkit

### 📚 Documentation

- Add README with overview and usage examples

### 🧪 Testing

- Use testify in pkg env tests

### ⚙️ Miscellaneous Tasks

- Add GoReleaser release workflow and config
