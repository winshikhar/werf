package file_reader

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/werf/werf/pkg/util"
)

func (r FileReader) isCommitFileExist(ctx context.Context, relPath string) (bool, error) {
	exist, err := r.manager.LocalGitRepo().IsCommitFileExists(ctx, r.manager.HeadCommit(), relPath)
	if err != nil {
		err := fmt.Errorf(
			"unable to check existence of file %s in the project git repo commit %s: %s",
			relPath, r.manager.HeadCommit(), err,
		)
		return false, err
	}

	return exist, nil
}

func (r FileReader) readCommitFile(ctx context.Context, relPath string, handleUncommittedChangesFunc func(context.Context, string) error) ([]byte, error) {
	fileRepoPath := filepath.ToSlash(relPath)

	repoData, err := r.manager.LocalGitRepo().ReadCommitFile(ctx, r.manager.HeadCommit(), fileRepoPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %q from the local git repo commit %s: %s", fileRepoPath, r.manager.HeadCommit(), err)
	}

	if handleUncommittedChangesFunc == nil {
		return repoData, nil
	}

	isDataIdentical, err := compareGitRepoFileWithProjectFile(repoData, r.manager.ProjectDir(), relPath)
	if err != nil {
		return nil, fmt.Errorf("error comparing repo file %q with the local project file: %s", fileRepoPath, err)
	}

	if !isDataIdentical {
		if err := handleUncommittedChangesFunc(ctx, relPath); err != nil {
			return nil, err
		}
	}

	return repoData, err
}

func compareGitRepoFileWithProjectFile(repoFileData []byte, projectDir, relPath string) (bool, error) {
	var localData []byte
	absPath := filepath.Join(projectDir, relPath)
	exist, err := util.FileExists(absPath)
	if err != nil {
		return false, fmt.Errorf("unable to check file existence: %s", err)
	} else if exist {
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
