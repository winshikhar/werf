package file_reader

import (
	"fmt"
	"io/ioutil"
	"os"
	pathPkg "path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"

	"github.com/werf/werf/pkg/util"
)

func (r FileReader) walkFilesWithGlob(dir, glob string) ([]string, error) {
	var result []string

	glob = filepath.FromSlash(glob)
	matchFunc := func(path string) (bool, error) {
		for _, glob := range []string{
			glob,
			pathPkg.Join(glob, "**", "*"),
		} {
			matched, err := doublestar.PathMatch(glob, path)
			if err != nil {
				return false, err // TODO
			}

			if matched {
				return true, nil
			}
		}

		return false, nil
	}

	err := r.walkFiles(dir, func(notResolvedPath string) error {
		matched, err := matchFunc(notResolvedPath)
		if err != nil {
			return fmt.Errorf("xxxx") // TODO
		}

		if matched {
			result = append(result, notResolvedPath)
		}

		return nil
	})

	return result, err
}

func (r FileReader) walkFiles(relDir string, fileFunc func(notResolvedPath string) error) error {
	exist, err := r.isDirectoryExist(relDir)
	if err != nil {
		return err
	}

	if !exist {
		return nil
	}

	resolveDir, err := r.ResolveFilePath(relDir)
	if err != nil {
		return fmt.Errorf("unable to resolve path %s", relDir)
	}

	absDirPath := filepath.Join(r.sharedContext.ProjectDir(), resolveDir)
	return filepath.Walk(absDirPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		notResolvedPath := util.GetRelativeToBaseFilepath(r.sharedContext.ProjectDir(), strings.Replace(path, resolveDir, relDir, 1))
		if f.Mode()&os.ModeSymlink == os.ModeSymlink {
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("eval symlink %s failed: %s", path, err)
			}

			linkStat, err := os.Lstat(link)
			if err != nil {
				return fmt.Errorf("lstat %s failed: %s", linkStat, err)
			}

			if linkStat.IsDir() {
				return r.walkFiles(notResolvedPath, fileFunc)
			}

			return fileFunc(notResolvedPath)
		}

		return fileFunc(notResolvedPath)
	})
}

func (r FileReader) checkAndReadFile(relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) ([]byte, error) {
	resolvedPath, accepted, err := r.ResolveAndValidateFilePath(relPath, isFileAcceptedFunc)
	if err != nil {
		return nil, err
	}

	if !accepted {
		return nil, skipReadingNotAcceptedFile
	}

	return r.readFile(resolvedPath)
}

func (r FileReader) ResolveAndValidateFilePath(relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) (string, bool, error) {
	validateFunc := func(resolvedPath string) error {
		if r.sharedContext.LooseGiterminism() {
			return nil
		}

		if r.sharedContext.IsFileInsideUninitializedSubmodule(relPath) {
			return skipReadingFileInsideUninitializedSubmodule
		}

		accepted, err := isFileAcceptedFunc(relPath)
		if err != nil {
			return err
		}

		if !accepted {
			return skipNotAcceptedFile
		}

		return nil
	}

	errHandleFunc := func(err error) (string, bool, error) {
		if err == skipReadingFileInsideUninitializedSubmodule || err == skipNotAcceptedFile {
			return "", false, nil
		}

		return "", false, err
	}

	if err := validateFunc(relPath); err != nil {
		return errHandleFunc(err)
	}

	resolvedPath, err := r.ResolveAndCheckFilePath(relPath, validateFunc)
	if err != nil {
		return errHandleFunc(err)
	}

	return resolvedPath, true, nil
}

// readFile resolves symlinks and returns the file data.
func (r FileReader) readFile(relPath string) ([]byte, error) {
	resolvedRelPath, err := r.ResolveFilePath(relPath)
	if err != nil {
		absPath := filepath.Join(r.sharedContext.ProjectDir(), relPath)
		return nil, fmt.Errorf("unable to resolve file path %s: %s", absPath, err)
	}

	absPath := filepath.Join(r.sharedContext.ProjectDir(), resolvedRelPath)
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s: %s", absPath, err)
	}

	return data, nil
}

// isDirectoryExist resolves symlinks and returns true if the resolved file is a directory.
func (r FileReader) isDirectoryExist(relPath string) (bool, error) {
	resolvedRelPath, err := r.ResolveFilePath(relPath)
	if err != nil {
		if err == FileNotFoundInProjectDirectoryErr {
			return false, nil
		}

		absPath := filepath.Join(r.sharedContext.ProjectDir(), relPath)
		return false, fmt.Errorf("unable to resolve file path %s: %s", absPath, err)
	}

	absPath := filepath.Join(r.sharedContext.ProjectDir(), resolvedRelPath)
	exist, err := util.DirExists(absPath)
	if err != nil {
		return false, fmt.Errorf("unable to check existence of directory %s: %s", absPath, err)
	}

	return exist, nil
}

// isRegularFileExist resolves symlinks and returns true if the resolved file is a regular file.
func (r FileReader) isRegularFileExist(relPath string) (bool, error) {
	resolvedRelPath, err := r.ResolveFilePath(relPath)
	if err != nil {
		if err == FileNotFoundInProjectDirectoryErr {
			return false, nil
		}

		absPath := filepath.Join(r.sharedContext.ProjectDir(), relPath)
		return false, fmt.Errorf("unable to resolve file path %s: %s", absPath, err)
	}

	absPath := filepath.Join(r.sharedContext.ProjectDir(), resolvedRelPath)
	exist, err := util.RegularFileExists(absPath)
	if err != nil {
		return false, fmt.Errorf("unable to check existence of file %s: %s", absPath, err)
	}

	return exist, nil
}

var (
	FileNotFoundInProjectDirectoryErr = fmt.Errorf("file not found in the project directory")
	TooManyLevelsOfSymbolicLinksErr   = fmt.Errorf("too many levels of symbolic links")
)

func (r FileReader) ResolveAndCheckFilePath(relPath string, checkFunc func(resolvedPath string) error) (string, error) {
	return r.resolveFilePath(relPath, 0, checkFunc)
}

func (r FileReader) ResolveFilePath(relPath string) (string, error) {
	return r.resolveFilePath(relPath, 0, nil)
}

func (r FileReader) resolveFilePath(relPath string, depth int, checkFunc func(resolvedPath string) error) (string, error) {
	fmt.Println(relPath, depth)

	if depth > 1000 {
		return "", TooManyLevelsOfSymbolicLinksErr
	}
	depth++

	// returns path if the corresponding file is File or Directory.
	//{
	//	absPath := filepath.Join(r.sharedContext.ProjectDir(), relPath)
	//	stat, err := os.Stat(absPath)
	//	if err != nil {
	//		if os.IsNotExist(err) || util.IsNotADirectoryError(err) {
	//			goto afterBlock
	//		}
	//
	//		return "", fmt.Errorf("unable to access file %s: %s", absPath, err)
	//	}
	//
	//	if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
	//		goto afterBlock
	//	}
	//
	//	if checkFunc != nil {
	//		if err := checkFunc(relPath); err != nil {
	//			return "", err
	//		}
	//	}
	//
	//	return relPath, nil
	//}

	//afterBlock:

	pathParts := util.SplitFilepath(relPath)
	pathPartsLen := len(pathParts)

	var resolvedPath string
	for ind := 0; ind < pathPartsLen; ind++ {
		isLastPathPart := ind == pathPartsLen-1
		pathToResolve := filepath.Join(resolvedPath, pathParts[ind])
		absPathToResolve := filepath.Join(r.sharedContext.ProjectDir(), pathToResolve)

		stat, err := os.Stat(absPathToResolve)
		if err != nil {
			if os.IsNotExist(err) || util.IsNotADirectoryError(err) {
				return "", FileNotFoundInProjectDirectoryErr
			}

			return "", fmt.Errorf("unable to access file %s: %s", absPathToResolve, err)
		}

		switch {
		case stat.Mode()&os.ModeSymlink == os.ModeSymlink:
			link, err := os.Readlink(absPathToResolve)
			if err != nil {
				return "", fmt.Errorf("unable to eval symlinks %s: %s", link, err)
			}

			resolvedLink := link
			if !filepath.IsAbs(link) {
				resolvedLink = filepath.Join(absPathToResolve, link)
			}

			if !util.IsSubpathOfBasePath(r.sharedContext.ProjectDir(), resolvedLink) {
				return "", FileNotFoundInProjectDirectoryErr
			}

			resolvedTarget, err := r.resolveFilePath(relPath, depth, checkFunc)
			if err != nil {
				return "", err
			}

			resolvedPath = resolvedTarget
		case !stat.IsDir() && !isLastPathPart:
			return "", FileNotFoundInProjectDirectoryErr
		default:
			resolvedPath = pathToResolve
		}
	}

	if checkFunc != nil {
		if err := checkFunc(resolvedPath); err != nil {
			return "", err
		}
	}

	return resolvedPath, nil
}
