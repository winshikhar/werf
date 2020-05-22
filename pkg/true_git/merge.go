package true_git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CreateTemporaryMergeCommitOptions struct {
	HasSubmodules bool
}

func CreateTemporaryMergeCommit(gitDir, workTreeCacheDir, fromCommit, intoCommit string, opts CreateTemporaryMergeCommitOptions) (string, error) {
	var resCommit string

	if err := withWorkTreeCacheLock(workTreeCacheDir, func() error {
		var err error

		gitDir, err = filepath.Abs(gitDir)
		if err != nil {
			return fmt.Errorf("bad git dir %s: %s", gitDir, err)
		}

		workTreeCacheDir, err = filepath.Abs(workTreeCacheDir)
		if err != nil {
			return fmt.Errorf("bad work tree cache dir %s: %s", workTreeCacheDir, err)
		}

		if opts.HasSubmodules {
			err := checkSubmoduleConstraint()
			if err != nil {
				return err
			}
		}

		if workTreeDir, err := prepareWorkTree(gitDir, workTreeCacheDir, intoCommit, opts.HasSubmodules); err != nil {
			return fmt.Errorf("unable to prepare worktree for commit %v: %s", intoCommit, err)
		} else {
			var err error
			var cmd *exec.Cmd
			var output *bytes.Buffer

			cmd = exec.Command(
				"git", "--git-dir", gitDir, "--work-tree", workTreeDir,
				"merge", "--no-edit", "--no-ff", fromCommit,
			)
			cmd.Dir = workTreeDir
			output = setCommandRecordingLiveOutput(cmd)
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("git merge %q failed: %s\n%s", strings.Join(append([]string{cmd.Path}, cmd.Args[1:]...), " "), err, output.String())
			}
			if debugMerge() {
				fmt.Printf("[DEBUG MERGE] %s\n%s\n", strings.Join(append([]string{cmd.Path}, cmd.Args[1:]...), " "), output)
			}

			cmd = exec.Command("git", "rev-parse", "HEAD")
			output = setCommandRecordingLiveOutput(cmd)
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("git merge failed: %s\n%s", err, output.String())
			}
			if debugMerge() {
				fmt.Printf("[DEBUG MERGE] %s\n%s\n", strings.Join(append([]string{cmd.Path}, cmd.Args[1:]...), " "), output)
			}

			resCommit = strings.TrimSpace(output.String())
		}

		return nil
	}); err != nil {
		return "", err
	}

	return resCommit, nil
}

func debugMerge() bool {
	return os.Getenv("WERF_TRUE_GIT_MERGE_DEBUG") == "1"
}
