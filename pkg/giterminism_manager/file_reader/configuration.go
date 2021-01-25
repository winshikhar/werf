package file_reader

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

func (r FileReader) configurationFilesGlob(ctx context.Context, dir, glob string, isFileAcceptedFunc func(relPath string) (bool, error), handleFileFunc func(relPath string, data []byte, err error) error) error {
	processedFiles := map[string]bool{}

	isFileProcessedFunc := func(relPath string) bool {
		return processedFiles[filepath.ToSlash(relPath)]
	}

	readFileBeforeHookFunc := func(relPath string) {
		processedFiles[filepath.ToSlash(relPath)] = true
	}

	readFileFunc := func(relPath string) ([]byte, error) {
		readFileBeforeHookFunc(relPath)
		return r.readFile(relPath)
	}

	readCommitFileWrapperFunc := func(relPath string) ([]byte, error) {
		readFileBeforeHookFunc(relPath)
		return r.checkAndReadCommitConfigurationFile(ctx, relPath)
	}

	fileRelPathListFromFS, err := r.walkFilesWithGlob(dir, glob)
	if err != nil {
		return err
	}
	fmt.Println("fs!!!!!!", fileRelPathListFromFS)

	if r.sharedContext.LooseGiterminism() {
		for _, relPath := range fileRelPathListFromFS {
			data, err := readFileFunc(relPath)
			if err := handleFileFunc(relPath, data, err); err != nil {
				return err
			}
		}

		return nil
	}

	fileRelPathListFromCommit, err := r.commitFilesGlob(ctx, dir, glob)
	if err != nil {
		return err
	}
	fmt.Println("commit!!!!!!", fileRelPathListFromCommit)

	var relPathListWithUncommittedFilesChanges []string
	for _, relPath := range fileRelPathListFromCommit {
		if accepted, err := isFileAcceptedFunc(relPath); err != nil {
			return err
		} else if accepted {
			continue
		}

		data, err := readCommitFileWrapperFunc(relPath)
		if err := handleFileFunc(relPath, data, err); err != nil {
			if isUncommittedFilesChangesError(err) {
				relPathListWithUncommittedFilesChanges = append(relPathListWithUncommittedFilesChanges, relPath)
				continue
			}

			return err
		}
	}

	if len(relPathListWithUncommittedFilesChanges) != 0 {
		return NewUncommittedFilesChangesError(relPathListWithUncommittedFilesChanges...)
	}

	var relPathListWithUncommittedFiles []string
	for _, relPath := range fileRelPathListFromFS {
		accepted, err := isFileAcceptedFunc(relPath)
		if err != nil {
			return err
		}

		if !accepted {
			if !isFileProcessedFunc(relPath) {
				relPathListWithUncommittedFiles = append(relPathListWithUncommittedFiles, relPath)
			}

			continue
		}

		data, err := readFileFunc(relPath)
		if err := handleFileFunc(relPath, data, err); err != nil {
			return err
		}
	}

	if len(relPathListWithUncommittedFiles) != 0 {
		return NewUncommittedFilesError(relPathListWithUncommittedFiles...)
	}

	return nil
}

func (r FileReader) checkConfigurationDirectoryExistence(ctx context.Context, relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) error {
	accepted, err := isFileAcceptedFunc(relPath)
	if err != nil {
		return err
	}

	shouldReadFromFS := r.sharedContext.LooseGiterminism() || accepted
	if !shouldReadFromFS {
		if exist, err := r.isCommitDirectoryExist(ctx, relPath); err != nil {
			return err
		} else if exist {
			return nil
		}
	}

	exist, err := r.isDirectoryExist(relPath)
	if err != nil {
		return err
	}

	if exist {
		if shouldReadFromFS {
			return nil
		} else {
			return NewUncommittedFilesError(relPath)
		}
	} else {
		if shouldReadFromFS {
			return NewFilesNotFoundInTheProjectDirectoryError(relPath)
		} else {
			return NewFilesNotFoundInTheProjectGitRepositoryError(relPath)
		}
	}
}

func (r FileReader) checkConfigurationFileExistence(ctx context.Context, relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) error {
	accepted, err := isFileAcceptedFunc(relPath)
	if err != nil {
		return err
	}

	shouldReadFromFS := r.sharedContext.LooseGiterminism() || accepted
	if !shouldReadFromFS {
		if exist, err := r.isCommitFileExist(ctx, relPath); err != nil {
			return err
		} else if exist {
			return nil
		}
	}

	exist, err := r.isRegularFileExist(relPath)
	if err != nil {
		return err
	}

	if exist {
		if shouldReadFromFS {
			return nil
		} else {
			_, err := r.ResolveAndCheckFilePath(relPath, func(resolvedPath string) error {
				fmt.Println(resolvedPath)

				exist, err := r.sharedContext.LocalGitRepo().IsTreeEntryExist(ctx, r.sharedContext.HeadCommit(), resolvedPath)
				if err != nil {
					return err
				}

				if exist {
					return nil
				} else {
					return NewUncommittedFilesError(relPath)
				}
			})

			if err != nil {
				return err
			}

			return NewUncommittedFilesError(relPath)
		}
	} else {
		if shouldReadFromFS {
			return NewFilesNotFoundInTheProjectDirectoryError(relPath)
		} else {
			return NewFilesNotFoundInTheProjectGitRepositoryError(relPath)
		}
	}
}

func (r FileReader) isConfigurationFileExistAnywhere(ctx context.Context, relPath string) (bool, error) {
	if exist, err := r.isCommitFileExist(ctx, relPath); err != nil {
		return false, err
	} else if !exist {
		return r.isRegularFileExist(relPath)
	} else {
		return true, nil
	}
}

func (r FileReader) isConfigurationDirectoryExistAnywhere(ctx context.Context, relPath string) (bool, error) {
	if exist, err := r.isCommitDirectoryExist(ctx, relPath); err != nil {
		return false, err
	} else if !exist {
		return r.isDirectoryExist(relPath)
	} else {
		return true, nil
	}
}

func (r FileReader) isConfigurationDirectoryExist(ctx context.Context, relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) (bool, error) {
	accepted, err := isFileAcceptedFunc(relPath)
	if err != nil {
		return false, err
	}

	if r.sharedContext.LooseGiterminism() || accepted {
		return r.isDirectoryExist(relPath)
	}

	return r.isCommitDirectoryExist(ctx, relPath)
}

func (r FileReader) isConfigurationFileExist(ctx context.Context, relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) (bool, error) {
	accepted, err := isFileAcceptedFunc(relPath)
	if err != nil {
		return false, err
	}

	if r.sharedContext.LooseGiterminism() || accepted {
		return r.isRegularFileExist(relPath)
	}

	return r.isCommitFileExist(ctx, relPath)
}

var skipReadingNotAcceptedFile = errors.New("skip reading not accepted file")
var skipReadingFileInsideUninitializedSubmodule = errors.New("skip reading file inside uninitialized submodule")
var skipNotAcceptedFile = errors.New("skip not accepted file") //TODO

func (r FileReader) checkAndReadConfigurationFile(ctx context.Context, relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) ([]byte, error) {
	data, err := r.checkAndReadFile(relPath, isFileAcceptedFunc)
	if err != nil {
		if err == skipReadingFileInsideUninitializedSubmodule || err == skipReadingNotAcceptedFile {
			goto readCommitFile
		}

		return nil, err
	}

	return data, nil

readCommitFile:
	return r.checkAndReadCommitConfigurationFile(ctx, relPath)
}
