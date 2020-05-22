package git_repo

import (
	"fmt"
	"io/ioutil"

	"github.com/flant/werf/pkg/testing/utils"

	"github.com/flant/werf/pkg/true_git"
	"github.com/flant/werf/pkg/werf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func GetPatchBetweenVirtualMergeCommits(workTreeDir string, v1FromCommit, v1IntoCommit, v2FromCommit, v2IntoCommit string) (string, error) {
	if gitRepo, err := OpenLocalRepo("virtual_merge_commit", workTreeDir); err != nil {
		return "", err
	} else {
		if v1Commit, err := gitRepo.CreateVirtualMergeCommit(v1FromCommit, v1IntoCommit); err != nil {
			return "", fmt.Errorf("unable to create virtual merge commit of %s and %s: %s", v1FromCommit, v1IntoCommit, err)
		} else if v2Commit, err := gitRepo.CreateVirtualMergeCommit(v2FromCommit, v2IntoCommit); err != nil {
			return "", fmt.Errorf("unable to create virtual merge commit of %s and %s: %s", v2FromCommit, v2IntoCommit, err)
		} else {
			fmt.Printf("v1Commit: %s\nv2Commit: %s\n", v1Commit, v2Commit)

			if patch, err := gitRepo.CreatePatch(PatchOptions{
				FromCommit:            v1Commit,
				ToCommit:              v2Commit,
				WithEntireFileContext: true,
				WithBinary:            true,
			}); err != nil {
				return "", fmt.Errorf("unable to create patch between %s and %s: %s", v1Commit, v2Commit, err)
			} else {
				if data, err := ioutil.ReadFile(patch.GetFilePath()); err != nil {
					return "", fmt.Errorf("error reading %s: %s", err)
				} else {
					return string(data), nil
				}
			}
		}
	}
}

var _ = Describe("GitRepo", func() {
	BeforeSuite(func() {
		Expect(werf.Init("", "")).To(Succeed())
		Expect(true_git.Init(true_git.Options{LiveGitOutput: true})).To(Succeed())
	})

	Context("", func() {
		AfterEach(func() {
		})

		It("should successfully build image using arbitrary ansible modules", func() {
			//Expect(werfBuild("general", liveexec.ExecCommandOptions{})).To(Succeed())

			Expect(utils.SetGitRepoState("git_repo_test/001-base1", "git_repo_test/repo", "base1")).To(Succeed())
			base1Commit := utils.SucceedCommandOutputString("git", "--work-tree", "git_repo_test/001-base1", "rev-parse", "HEAD")

			utils.SucceedCommandOutputString("git", "--work-tree", "git_repo_test/001-base1", "checkout", "-b", "change")
			Expect(utils.SetGitRepoState("git_repo_test/002-change1", "git_repo_test/repo", "change1")).To(Succeed())
			change1Commit := utils.SucceedCommandOutputString("git", "--work-tree", "git_repo_test/002-change1", "rev-parse", "HEAD")

			utils.SucceedCommandOutputString("git", "--work-tree", "git_repo_test/002-change1", "checkout", "master")
			Expect(utils.SetGitRepoState("git_repo_test/003-base2", "git_repo_test/repo", "base2")).To(Succeed())
			base2Commit := utils.SucceedCommandOutputString("git", "--work-tree", "git_repo_test/003-base2", "rev-parse", "HEAD")

			utils.SucceedCommandOutputString("git", "--work-tree", "git_repo_test/003-base2", "checkout", "change")
			Expect(utils.SetGitRepoState("git_repo_test/004-change2", "git_repo_test/repo", "change2")).To(Succeed())
			change2Commit := utils.SucceedCommandOutputString("git", "--work-tree", "git_repo_test/004-change2", "rev-parse", "HEAD")

			if patch, err := GetPatchBetweenVirtualMergeCommits("git_repo_test/004-change2", change1Commit, base1Commit, change2Commit, base2Commit); err != nil {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(patch).To(Equal(`
`))
			}
		})
	})
})
