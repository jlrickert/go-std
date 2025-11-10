package project

import (
	"context"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jlrickert/cli-toolkit/mylog"
	"github.com/jlrickert/cli-toolkit/toolkit"
)

// findGitRoot attempts to use the git CLI to determine the repository top-level
// directory starting from 'start'. If that fails (git not available, not a git
// worktree, or command error), it falls back to the original upward filesystem
// search for a .git entry.
func FindGitRoot(ctx context.Context, start string) string {
	lg := mylog.LoggerFromContext(ctx)

	// Normalize start to a directory (in case a file path was passed).
	if fi, err := toolkit.Stat(ctx, start, false); err == nil && !fi.IsDir() {
		start = filepath.Dir(start)
	}

	// First, try using git itself to find the top-level directory. Using `-C`
	// makes git operate relative to the provided path.
	args := []string{"-C", start, "rev-parse", "--show-toplevel"}
	if out, err := exec.CommandContext(ctx, "git", args...).Output(); err == nil {
		if p := strings.TrimSpace(string(out)); p != "" {
			lg.Log(
				ctx,
				slog.LevelDebug,
				"git rev-parse succeeded",
				slog.String("root", p),
			)
			return p
		}
		lg.Log(ctx, slog.LevelDebug, "git rev-parse returned empty output")
	} else {
		lg.Log(
			ctx,
			slog.LevelWarn,
			"git rev-parse failed, falling back",
			slog.String("start", start),
			slog.Any("error", err),
		)
	}

	// Fallback: walk upwards looking for a .git entry (dir or file).
	p := start
	for {
		gitPath := filepath.Join(p, ".git")
		if fi, err := toolkit.Stat(ctx, gitPath, false); err == nil {
			// .git can be a dir (normal repo) or a file (worktree / submodule).
			if fi.IsDir() || fi.Mode().IsRegular() {
				lg.Log(ctx, slog.LevelDebug, "found .git entry", slog.String("root", p))
				return p
			}
		}
		parent := filepath.Dir(p)
		if parent == p {
			// reached filesystem root
			break
		}
		p = parent
	}
	lg.Log(ctx, slog.LevelDebug, "git root not found", slog.String("start", start))
	return ""
}
