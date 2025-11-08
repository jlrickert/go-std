package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	std "github.com/jlrickert/go-std/pkg"
)

// Project holds paths and configuration roots for a repository-backed
// project. Root is the repository root. Other roots default to platform
// user-scoped locations when not provided.
type Project struct {
	Appname string

	// Root is the path to the root of the project.
	Root string

	// configRoot is the base directory for user configuration files.
	configRoot string

	// stateRoot holds transient state files for the project.
	stateRoot string

	// dataRoot is for programmatically managed data shipped with the program.
	dataRoot string

	// cacheRoot is for cache artifacts.
	cacheRoot string

	// localConfigRoot is the repo-local override location
	localConfigRoot string
}

type ProjectOption = func(ctx context.Context, p *Project)

// WithStateRoot sets the state root path on the Project.
func WithStateRoot(path string) ProjectOption {
	return func(ctx context.Context, opt *Project) {
		opt.stateRoot = path
	}
}

// WithConfigRoot sets the config root path on the Project.
func WithConfigRoot(path string) ProjectOption {
	return func(ctx context.Context, opt *Project) {
		opt.configRoot = path
	}
}

// WithDataRoot sets the data root path on the Project.
func WithDataRoot(path string) ProjectOption {
	return func(ctx context.Context, opt *Project) {
		opt.dataRoot = path
	}
}

// WithCacheRoot sets the cache root path on the Project.
func WithCacheRoot(path string) ProjectOption {
	return func(ctx context.Context, opt *Project) {
		opt.cacheRoot = path
	}
}

// WithRoot sets the repository root on the Project.
func WithRoot(path string) ProjectOption {
	return func(ctx context.Context, opt *Project) {
		opt.Root = path
	}
}

// WithAutoRootDetect returns an option that sets Root by detecting the
// repository top-level directory using the Env from the provided context.
// If detection fails the option leaves Root unchanged.
func WithAutoRootDetect() ProjectOption {
	return func(ctx context.Context, p *Project) {
		env := std.EnvFromContext(ctx)
		wd, err := env.Getwd()
		if err != nil {
			// leave Root unchanged when we cannot determine working dir
			return
		}
		root := FindGitRoot(ctx, wd)
		p.Root = root
	}
}

// ConfigRoot returns the configured config root. When not set it derives a
// sensible default using the provided context and the Project Appname.
func (p *Project) ConfigRoot(ctx context.Context) (string, error) {
	if p == nil {
		return "", fmt.Errorf("nil project")
	}
	if p.configRoot != "" {
		return std.AbsPath(ctx, p.configRoot), nil
	}
	path, err := std.UserConfigPath(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(path, p.Appname), nil
}

// DataRoot returns the configured data root or a platform default joined with
// the Project Appname.
func (p *Project) DataRoot(ctx context.Context) (string, error) {
	if p == nil {
		return "", fmt.Errorf("nil project")
	}
	if p.dataRoot != "" {
		return p.dataRoot, nil
	}
	path, err := std.UserDataPath(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(path, p.Appname), nil
}

// StateRoot returns the configured state root or a platform default joined
// with the Project Appname.
func (p *Project) StateRoot(ctx context.Context) (string, error) {
	if p == nil {
		return "", fmt.Errorf("nil project")
	}
	if p.stateRoot != "" {
		return p.stateRoot, nil
	}
	path, err := std.UserStatePath(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(path, p.Appname), nil
}

// CacheRoot returns the configured cache root or a platform default joined
// with the Project Appname.
func (p *Project) CacheRoot(ctx context.Context) (string, error) {
	if p == nil {
		return "", fmt.Errorf("nil project")
	}
	if p.cacheRoot != "" {
		return p.cacheRoot, nil
	}
	path, err := std.UserCachePath(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(path, p.Appname), nil
}

// LocalConfigRoot returns the repo-local config root. When not explicitly set
// it defaults to "<Root>/.<appname>".
func (p *Project) LocalConfigRoot(ctx context.Context) (string, error) {
	if p == nil {
		return "", fmt.Errorf("nil project")
	}
	if p.localConfigRoot != "" {
		return p.localConfigRoot, nil
	}
	if p.Root == "" {
		return "", fmt.Errorf("project root not set")
	}
	return filepath.Join(p.Root, "."+p.Appname), nil
}

// NewProject constructs a Project and fills missing roots using platform
// defaults derived from the provided context.
//
// Behavior:
//   - If an option sets a value it is used as-is.
//   - If Root is not set it is inferred from Env.Getwd().
//   - ConfigRoot, DataRoot, StateRoot and CacheRoot use the corresponding
//     user-scoped platform paths and are joined with DefaultAppName.
func NewProject(ctx context.Context, appname string, opts ...ProjectOption) (*Project, error) {
	p := &Project{Appname: appname}

	for _, f := range opts {
		f(ctx, p)
	}

	env := std.EnvFromContext(ctx)

	if p.Root == "" {
		wd, err := env.Getwd()
		if err != nil {
			return p, fmt.Errorf("unable to infer project: %w", err)
		}
		p.Root = wd
	}

	if p.configRoot == "" {
		if path, err := std.UserConfigPath(ctx); err != nil {
			return nil, fmt.Errorf(
				"unable to find user config path: %w",
				os.ErrNotExist,
			)
		} else {
			p.configRoot = filepath.Join(path, p.Appname)
		}
	}

	if p.dataRoot == "" {
		if path, err := std.UserDataPath(ctx); err != nil {
			return nil, fmt.Errorf(
				"unable to find user data path: %w",
				os.ErrNotExist,
			)
		} else {
			p.dataRoot = filepath.Join(path, p.Appname)
		}
	}

	if p.stateRoot == "" {
		if path, err := std.UserStatePath(ctx); err != nil {
			return nil, fmt.Errorf(
				"unable to find user state root: %w",
				os.ErrNotExist,
			)
		} else {
			p.stateRoot = filepath.Join(path, p.Appname)
		}
	}

	if p.cacheRoot == "" {
		if path, err := std.UserCachePath(ctx); err != nil {
			return nil, fmt.Errorf(
				"unable to find user cache root: %w",
				os.ErrNotExist,
			)
		} else {
			p.cacheRoot = filepath.Join(path, p.Appname)
		}
	}

	return p, nil
}
