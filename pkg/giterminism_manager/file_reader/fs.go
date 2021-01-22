package file_reader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/werf/werf/pkg/util"
)

// Glob returns the hash of regular files and their contents for the paths that are matched pattern
// This function follows only symlinks pointed to a regular file (not to a directory)
func (r FileReader) filesGlob(pattern string) ([]string, error) {
	var result []string
	err := util.WalkByPattern(r.sharedContext.ProjectDir(), pattern, func(path string, s os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if s.IsDir() {
			return nil
		}

		var filePath string
		if s.Mode()&os.ModeSymlink == os.ModeSymlink {
			link, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("eval symlink %s failed: %s", path, err)
			}

			linkStat, err := os.Lstat(link)
			if err != nil {
				return fmt.Errorf("lstat %s failed: %s", linkStat, err)
			}

			if linkStat.IsDir() {
				return nil
			}

			filePath = link
		} else {
			filePath = path
		}

		if util.IsSubpathOfBasePath(r.sharedContext.ProjectDir(), filePath) {
			relPath := util.GetRelativeToBaseFilepath(r.sharedContext.ProjectDir(), filePath)
			result = append(result, relPath)
		} else {
			return fmt.Errorf("unable to handle the file %s which is located outside the project directory", filePath)
		}

		return nil
	})

	return result, err
}

func (r FileReader) checkAndReadFile(relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) ([]byte, error) {
	accepted, err := r.checkFilePath(relPath, isFileAcceptedFunc)
	if err != nil {
		return nil, err
	}

	if !accepted {
		return nil, skipReadingNotAcceptedFile
	}

	return r.readFile(relPath)
}

func (r FileReader) checkFilePath(relPath string, isFileAcceptedFunc func(relPath string) (bool, error)) (bool, error) {
	if r.sharedContext.LooseGiterminism() {
		return true, nil
	}

	_, err := r.CheckAndResolveFilePath(relPath, 0, func(resolvedPath string) error {
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
	})
	if err != nil {
		if err == skipReadingFileInsideUninitializedSubmodule || err == skipNotAcceptedFile {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// readFile resolves symlinks and returns the file data.
func (r FileReader) readFile(relPath string) ([]byte, error) {
	resolvedRelPath, err := r.ResolveFilePath(relPath, 0)
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
	resolvedRelPath, err := r.ResolveFilePath(relPath, 0)
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
	resolvedRelPath, err := r.ResolveFilePath(relPath, 0)
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

func (r FileReader) CheckAndResolveFilePath(relPath string, depth int, checkFunc func(resolvedPath string) error) (string, error) {
	path, err := r.resolveFilePath(relPath, depth, checkFunc)
	fmt.Printf("%q %q %q %q\n", "check and resolve", relPath, path, err)
	return path, err
}

func (r FileReader) ResolveFilePath(relPath string, depth int) (string, error) {
	path, err := r.resolveFilePath(relPath, depth, nil)
	fmt.Printf("%q %q %q %q\n", "resolve", relPath, path, err)
	return path, err
}

func (r FileReader) resolveFilePath(relPath string, depth int, checkFunc func(resolvedPath string) error) (string, error) {
	if depth > 1000 {
		return "", TooManyLevelsOfSymbolicLinksErr
	}
	depth++

	// returns path if the corresponding file is File or Directory.
	{
		absPath := filepath.Join(r.sharedContext.ProjectDir(), relPath)
		exist, err := util.FileExists(absPath)
		if err != nil {
			return "", fmt.Errorf("unable to check existence of file %s: %s", absPath, err)
		}

		if checkFunc != nil {
			if err := checkFunc(relPath); err != nil {
				return "", err
			}
		}

		if exist {
			return relPath, nil
		}
	}

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

			resolvedTarget, err := r.ResolveFilePath(relPath, 0)
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
