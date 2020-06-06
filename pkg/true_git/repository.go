package true_git

import (
	"github.com/go-git/go-git/v5"
)

func GitOpenWithCustomWorktreeDir(gitDir, worktreeDir string) (*git.Repository, error) {
	return git.PlainOpen(worktreeDir)

	//worktreeFilesystem := osfs.New(worktreeDir)
	//fmt.Printf("-- GitOpenWithCustomWorktreeDir gitDir=%s worktreeDir=%s\n", gitDir, worktreeDir)
	//
	//var gitDirForWorkTree string
	//workTreeDotGitFilePath := filepath.Join(worktreeDir, ".git")
	//if data, err := ioutil.ReadFile(workTreeDotGitFilePath); err != nil {
	//	return nil, fmt.Errorf("error reading %s: %s", workTreeDotGitFilePath, err)
	//} else {
	//	gitDirForWorkTree = strings.TrimSpace(strings.TrimPrefix(string(data), "gitdir: "))
	//}
	//
	//storage := filesystem.NewStorage(osfs.New(gitDirForWorkTree), cache.NewObjectLRUDefault())
	//
	//if cfg, err := storage.Config(); err != nil {
	//	return nil, err
	//} else {
	//	fmt.Printf("####### Config.Submodules -> %#v\n", cfg.Submodules)
	//}
	//
	//if moduleStorage, err := storage.Module("modules/jq"); err != nil {
	//	return nil, err
	//} else {
	//	fmt.Printf("####### moduleStorage -> %#v\n", moduleStorage)
	//	moduleStorage.Config()
	//}
	//
	//return git.Open(storage, worktreeFilesystem)
}
