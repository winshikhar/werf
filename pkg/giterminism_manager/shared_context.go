package giterminism_manager

import (
	"github.com/werf/werf/pkg/git_repo"
)

type sharedContext struct {
	projectDir       string
	headCommit       string
	localGitRepo     git_repo.Local
	looseGiterminism bool
	devMode          bool
}

type NewSharedContextOptions struct {
	looseGiterminism bool
	devMode          bool
}

func NewSharedContext(projectDir string, localGitRepo git_repo.Local, headCommit string, options NewSharedContextOptions) (*sharedContext, error) {
	c := &sharedContext{
		projectDir:       projectDir,
		headCommit:       headCommit,
		localGitRepo:     localGitRepo,
		looseGiterminism: options.looseGiterminism,
		devMode:          options.devMode,
	}

	return c, nil
}

func (c *sharedContext) ProjectDir() string {
	return c.projectDir
}

func (c *sharedContext) HeadCommit() string {
	return c.headCommit
}

func (c *sharedContext) LocalGitRepo() *git_repo.Local {
	return &c.localGitRepo
}

func (c *sharedContext) LooseGiterminism() bool {
	return c.looseGiterminism
}

func (c *sharedContext) DevMode() bool {
	return c.devMode
}
