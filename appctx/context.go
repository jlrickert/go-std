package appctx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlrickert/cli-toolkit/toolkit"
)

// AppContext holds paths and configuration roots for a repository-backed app
// context. Root is the repository root. Other roots default to platform
// user-scoped locations when not provided.
type AppContext struct {
	Appname string

	// Root is the path to the root of the context.
	Root string

	// configRoot is the base directory for user configuration files.
	ConfigRoot string

	// stateRoot holds transient state files for the context.
	StateRoot string

	// dataRoot is for programmatically managed data shipped with the program.
	DataRoot string

	// cacheRoot is for cache artifacts.
	CacheRoot string

	// localConfigRoot is the repo-local override location
	LocalConfigRoot string
}

func NewGitAppContext(ctx context.Context, appname string) (*AppContext, error) {
	env := toolkit.EnvFromContext(ctx)
	cwd, err := env.Getwd()
	if err != nil {
		return nil, err
	}
	root := FindGitRoot(ctx, cwd)
	aCtx, err := NewAppContext(ctx, appname)
	aCtx.Root = root
	return aCtx, err
}

// NewAppContext constructs a app context and fills missing roots using platform
// defaults derived from the provided context.
//
// Behavior:
//   - If an option sets a value it is used as-is.
//   - If Root is not set it is inferred from Env.Getwd().
//   - ConfigRoot, DataRoot, StateRoot and CacheRoot use the corresponding
//     user-scoped platform paths and are joined with DefaultAppName.
func NewAppContext(ctx context.Context, appname string) (*AppContext, error) {
	p := &AppContext{Appname: appname}

	env := toolkit.EnvFromContext(ctx)

	wd, err := env.Getwd()
	if err != nil {
		return p, fmt.Errorf("unable to infer app context: %w", err)
	}
	p.Root = wd

	if path, err := toolkit.UserConfigPath(ctx); err != nil {
		return nil, fmt.Errorf(
			"unable to find user config path: %w",
			os.ErrNotExist,
		)
	} else {
		p.ConfigRoot = filepath.Join(path, p.Appname)
	}

	if path, err := toolkit.UserDataPath(ctx); err != nil {
		return nil, fmt.Errorf(
			"unable to find user data path: %w",
			os.ErrNotExist,
		)
	} else {
		p.DataRoot = filepath.Join(path, p.Appname)
	}

	if path, err := toolkit.UserStatePath(ctx); err != nil {
		return nil, fmt.Errorf(
			"unable to find user state root: %w",
			os.ErrNotExist,
		)
	} else {
		p.StateRoot = filepath.Join(path, p.Appname)
	}

	if path, err := toolkit.UserCachePath(ctx); err != nil {
		return nil, fmt.Errorf(
			"unable to find user cache root: %w",
			os.ErrNotExist,
		)
	} else {
		p.CacheRoot = filepath.Join(path, p.Appname)
	}

	p.LocalConfigRoot = filepath.Join(p.Root, "."+appname)

	return p, nil
}
