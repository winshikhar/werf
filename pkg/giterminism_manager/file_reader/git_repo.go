package file_reader

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
)

func (r FileReader) isCommitFileExist(ctx context.Context, relPath string) (bool, error) {
	return r.sharedContext.LocalGitRepo().IsCommitFileExist(ctx, r.sharedContext.HeadCommit(), relPath)
}

func (r FileReader) isCommitDirectoryExist(ctx context.Context, relPath string) (bool, error) {
	return r.sharedContext.LocalGitRepo().IsCommitDirectoryExist(ctx, r.sharedContext.HeadCommit(), relPath)
}

func (r FileReader) checkAndReadCommitConfigurationFile(ctx context.Context, relPath string) ([]byte, error) {
	if err := r.checkCommitFilePath(ctx, relPath); err != nil {
		return nil, err
	}

	return r.readCommitFile(ctx, relPath)
}

func (r FileReader) checkCommitFilePath(ctx context.Context, relPath string) error {
	_, err := r.sharedContext.LocalGitRepo().CheckAndResolveCommitFilePath(ctx, r.sharedContext.HeadCommit(), relPath, func(resolvedRelPath string) error {
		if !r.sharedContext.IsFileInsideUninitializedSubmodule(resolvedRelPath) {
			fileChanged, err := r.sharedContext.IsWorktreeFileModified(ctx, resolvedRelPath)
			if err != nil {
				return err
			}

			if fileChanged {
				return NewUncommittedFilesChangesError(resolvedRelPath)
			}
		}

		return nil
	})

	return err
}

func (r FileReader) readCommitFile(ctx context.Context, relPath string) ([]byte, error) {
	return r.sharedContext.LocalGitRepo().ReadCommitFile(ctx, r.sharedContext.HeadCommit(), relPath)
}

func (r FileReader) commitFilesGlob(ctx context.Context, pattern string) ([]string, error) {
	var result []string

	commitPathList, err := r.sharedContext.LocalGitRepo().GetCommitFilePathList(ctx, r.sharedContext.HeadCommit())
	if err != nil {
		return nil, fmt.Errorf("unable to get files list from local git repo: %s", err)
	}

	pattern = filepath.ToSlash(pattern)
	for _, relFilepath := range commitPathList {
		relPath := filepath.ToSlash(relFilepath)
		if matched, err := doublestar.Match(pattern, relPath); err != nil {
			return nil, err
		} else if matched {
			result = append(result, relPath)
		}
	}

	return result, nil
}
