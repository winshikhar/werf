package giterminism_manager

import "github.com/werf/werf/pkg/git_repo"

type sharedContext struct {
	projectDir       string
	headCommit       string
	localGitRepo     git_repo.Local
	looseGiterminism bool
	devMode          bool
}

func (s *sharedContext) ProjectDir() string {
	return s.projectDir
}

func (s *sharedContext) HeadCommit() string {
	return s.headCommit
}

func (s *sharedContext) LocalGitRepo() *git_repo.Local {
	return &s.localGitRepo
}

func (s *sharedContext) LooseGiterminism() bool {
	return s.looseGiterminism
}

func (s *sharedContext) DevMode() bool {
	return s.devMode
}
