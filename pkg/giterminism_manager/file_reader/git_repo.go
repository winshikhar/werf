package file_reader

import (
	"context"
	"fmt"
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
	_, err := r.sharedContext.LocalGitRepo().ResolveAndCheckCommitFilePath(ctx, r.sharedContext.HeadCommit(), relPath, func(resolvedRelPath string) error {
		fmt.Println("checkCommitFilePath", resolvedRelPath)

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

func (r FileReader) commitFilesGlob(ctx context.Context, dir string, pattern string) ([]string, error) {
	return r.sharedContext.LocalGitRepo().WalkCommitFilesWithGlob(ctx, r.sharedContext.HeadCommit(), dir, pattern)
}
