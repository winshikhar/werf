package giterminism_manager

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/werf/werf/pkg/git_repo"
	"github.com/werf/werf/pkg/git_repo/status"
	"github.com/werf/werf/pkg/path_matcher"
	"github.com/werf/werf/pkg/util"
)

type sharedContext struct {
	projectDir       string
	headCommit       string
	localGitRepo     git_repo.Local
	looseGiterminism bool
	devMode          bool

	statusResult *status.Result
}

type NewSharedContextOptions struct {
	looseGiterminism bool
	devMode          bool
}

func NewSharedContext(ctx context.Context, projectDir string, localGitRepo git_repo.Local, headCommit string, options NewSharedContextOptions) (*sharedContext, error) {
	c := &sharedContext{
		projectDir:       projectDir,
		headCommit:       headCommit,
		localGitRepo:     localGitRepo,
		looseGiterminism: options.looseGiterminism,
		devMode:          options.devMode,
	}

	statusResult, err := localGitRepo.Status(ctx, path_matcher.NewSimplePathMatcher("", []string{}, true))
	if err != nil {
		return nil, fmt.Errorf("unable to get status for the project git repository: %s", err)
	}

	c.statusResult = statusResult

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

func (c *sharedContext) IsFileInsideUninitializedSubmodule(relPath string) bool {
	for _, submodulePath := range c.statusResult.UninitializedSubmodulePathList() {
		if util.IsSubpathOfBasePath(submodulePath, relPath) {
			return true
		}
	}

	return false
}

func (c *sharedContext) IsWorktreeFileModified(ctx context.Context, relPath string) (bool, error) {
	changed := c.statusResult.IsFileChanged(relPath, status.FilterOptions{WorktreeOnly: c.devMode})

	// https://github.com/go-git/go-git/issues/227
	if changed && runtime.GOOS == "windows" {
		commitFileData, err := c.localGitRepo.ReadCommitFile(ctx, c.headCommit, relPath)
		if err != nil {
			return false, err
		}

		isDataIdentical, err := compareGitRepoFileWithProjectFile(c.projectDir, relPath, commitFileData)
		if err != nil {
			return false, fmt.Errorf("unable to compare the project git repository file '%s' with worktree file: %s", relPath, err)
		}

		changed = !isDataIdentical
	}

	return changed, nil
}

func compareGitRepoFileWithProjectFile(projectDir, relPath string, repoFileData []byte) (bool, error) {
	absPath := filepath.Join(projectDir, relPath)
	exist, err := util.FileExists(absPath)
	if err != nil {
		return false, fmt.Errorf("unable to check file existence: %s", err)
	}

	var localData []byte
	if exist {
		localData, err = ioutil.ReadFile(absPath)
		if err != nil {
			return false, fmt.Errorf("unable to read file: %s", err)
		}
	}

	isDataIdentical := bytes.Equal(repoFileData, localData)
	localDataWithForcedUnixLineBreak := bytes.ReplaceAll(localData, []byte("\r\n"), []byte("\n"))
	if !isDataIdentical {
		isDataIdentical = bytes.Equal(repoFileData, localDataWithForcedUnixLineBreak)
	}

	return isDataIdentical, nil
}
