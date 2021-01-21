package true_git

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/werf/werf/pkg/util"
)

func GetWorkTreeByGitDir(gitDir string) (bool, string, error) {
	if filepath.Base(gitDir) == git.GitDirName {
		return true, filepath.Dir(gitDir), nil
	}
	return false, "", nil
}

func UpwardLookupAndVerifyGitDir(lookupPath string) (bool, string, error) {
	lookupPath = util.GetAbsoluteFilepath(lookupPath)

	for {
		dotGitPath := filepath.Join(lookupPath, git.GitDirName)

		if info, err := os.Stat(dotGitPath); os.IsNotExist(err) {
			if lookupPath != filepath.Dir(lookupPath) {
				lookupPath = filepath.Dir(lookupPath)
				continue
			}
		} else if err != nil {
			return false, "", fmt.Errorf("error accessing %q: %s", dotGitPath, err)
		} else if info.IsDir() {
			if isValid, err := IsValidGitDir(dotGitPath); err != nil {
				return false, "", err
			} else if isValid {
				return true, dotGitPath, nil
			}
		} else if gitDirLink, err := ReadGitDirLinkFromFile(dotGitPath); err != nil {
			return false, "", fmt.Errorf("error reading %q: %s", dotGitPath, err)
		} else if found, upwardGitDir, err := UpwardLookupAndVerifyNonSubmoduleGitDir(gitDirLink); err != nil {
			return false, "", err
		} else if found {
			return true, upwardGitDir, nil
		}

		break
	}

	return false, "", nil
}

func UpwardLookupAndVerifyNonSubmoduleGitDir(gitDir string) (bool, string, error) {
	if isValid, err := IsValidGitDir(gitDir); err != nil {
		return false, "", err
	} else if !isValid {
		return false, "", nil
	}

	parentDir := filepath.Dir(gitDir)
	parentBaseName := filepath.Base(parentDir)
	if parentBaseName == "modules" {
		if found, res, err := UpwardLookupAndVerifyNonSubmoduleGitDir(filepath.Dir(parentDir)); err != nil {
			return false, "", err
		} else if found {
			return found, res, nil
		} else {
			return true, gitDir, nil
		}
	} else {
		return true, gitDir, nil
	}
}

func IsValidGitDir(gitDir string) (bool, error) {
	gitArgs := []string{"--git-dir", gitDir, "rev-parse"}

	cmd := exec.Command("git", gitArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.HasPrefix(string(output), "fatal: not a git repository: ") {
			return false, nil
		}
		return false, fmt.Errorf("%v failed: %s:\n%s", strings.Join(append([]string{"git"}, gitArgs...), " "), err, output)
	}

	return true, nil
}

func ReadGitDirLinkFromFile(dotGitFilePath string) (string, error) {
	if b, err := ioutil.ReadFile(dotGitFilePath); err != nil {
		return "", fmt.Errorf("error reading %q: %s", dotGitFilePath, err)
	} else {
		// NOTE: this simple parser has been taken from the go-git: https://github.com/go-git/go-git/blob/master/repository.go#L341

		line := string(b)
		const prefix = "gitdir: "
		if !strings.HasPrefix(line, prefix) {
			return "", fmt.Errorf(".git file has no %s prefix", prefix)
		}

		gitdir := strings.Split(line[len(prefix):], "\n")[0]
		gitdir = strings.TrimSpace(gitdir)

		if filepath.IsAbs(gitdir) {
			return gitdir, nil
		}

		return filepath.Join(filepath.Dir(dotGitFilePath), gitdir), nil
	}
}
