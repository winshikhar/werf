package main

import (
	"fmt"
	"os"

	"github.com/flant/werf/pkg/true_git"

	"github.com/flant/werf/pkg/werf"

	"github.com/flant/werf/pkg/git_repo"
)

func do(v1FromCommit, v1IntoCommit, v2FromCommit, v2IntoCommit string) error {
	if gitRepo, err := git_repo.OpenLocalRepo("virtual_merge_commit", "."); err != nil {
		return err
	} else {

		if patch, err := gitRepo.CreatePatchBetweenVirtualCommits(v1FromCommit, v1IntoCommit, v2FromCommit, v2IntoCommit); err != nil {
			return err
		} else {
			fmt.Printf("THE PATCH:\n%s\n", patch)
		}
	}
	return nil
}

func main() {
	fmt.Printf("Virtual merge commit test BEGIN\n")

	if err := werf.Init("", ""); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	if err := true_git.Init(true_git.Options{LiveGitOutput: true}); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	if err := do(os.Args[1], os.Args[2], os.Args[3], os.Args[4]); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Virtual merge commit test SUCCEEDED\n")
}
