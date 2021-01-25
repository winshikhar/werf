package git_repo

import (
	"context"
	"fmt"
	"github.com/bmatcuk/doublestar"
	pathPkg "path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/filemode"

	"github.com/werf/logboek"

	"github.com/werf/werf/pkg/git_repo/status"
	"github.com/werf/werf/pkg/path_matcher"
	"github.com/werf/werf/pkg/true_git"
	"github.com/werf/werf/pkg/true_git/ls_tree"
	"github.com/werf/werf/pkg/util"
)

var (
	ErrLocalRepositoryNotExists     = git.ErrRepositoryNotExists
	EntryNotFoundInRepoErr          = fmt.Errorf("entry not found in the repo")
	TooManyLevelsOfSymbolicLinksErr = fmt.Errorf("too many levels of symbolic links")
)

type Local struct {
	Base
	Path   string
	GitDir string

	headCommit string
}

func OpenLocalRepo(name, path string, dev bool) (l Local, err error) {
	_, err = git.PlainOpenWithOptions(path, &git.PlainOpenOptions{EnableDotGitCommonDir: true})
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return l, ErrLocalRepositoryNotExists
		}

		return l, err
	}

	gitDir, err := true_git.GetRealRepoDir(filepath.Join(path, ".git"))
	if err != nil {
		return l, fmt.Errorf("unable to get real git repo dir for %s: %s", path, err)
	}

	l, err = newLocal(name, path, gitDir)
	if err != nil {
		return l, err
	}

	if dev {
		devHeadCommit, err := true_git.SyncDevBranchWithStagedFiles(
			context.Background(),
			l.GitDir,
			l.getRepoWorkTreeCacheDir(l.getRepoID()),
			l.headCommit,
		)
		if err != nil {
			return l, err
		}

		l.headCommit = devHeadCommit
	}

	return l, nil
}

func newLocal(name, path, gitDir string) (l Local, err error) {
	headCommit, err := getHeadCommit(path)
	if err != nil {
		return l, fmt.Errorf("unable to get git repo head commit: %s", err)
	}

	l = Local{
		Base:       Base{Name: name},
		Path:       path,
		GitDir:     gitDir,
		headCommit: headCommit,
	}

	return l, nil
}

func (repo *Local) PlainOpen() (*git.Repository, error) {
	return git.PlainOpen(repo.Path)
}

func (repo *Local) SyncWithOrigin(ctx context.Context) error {
	isShallow, err := repo.IsShallowClone()
	if err != nil {
		return fmt.Errorf("check shallow clone failed: %s", err)
	}

	remoteOriginUrl, err := repo.RemoteOriginUrl(ctx)
	if err != nil {
		return fmt.Errorf("get remote origin failed: %s", err)
	}

	if remoteOriginUrl == "" {
		return fmt.Errorf("git remote origin was not detected")
	}

	return logboek.Context(ctx).Default().LogProcess("Syncing origin branches and tags").DoError(func() error {
		fetchOptions := true_git.FetchOptions{
			Prune:     true,
			PruneTags: true,
			Unshallow: isShallow,
			RefSpecs:  map[string]string{"origin": "+refs/heads/*:refs/remotes/origin/*"},
		}

		if err := true_git.Fetch(ctx, repo.Path, fetchOptions); err != nil {
			return fmt.Errorf("fetch failed: %s", err)
		}

		return nil
	})
}

func (repo *Local) FetchOrigin(ctx context.Context) error {
	isShallow, err := repo.IsShallowClone()
	if err != nil {
		return fmt.Errorf("check shallow clone failed: %s", err)
	}

	remoteOriginUrl, err := repo.RemoteOriginUrl(ctx)
	if err != nil {
		return fmt.Errorf("get remote origin failed: %s", err)
	}

	if remoteOriginUrl == "" {
		return fmt.Errorf("git remote origin was not detected")
	}

	return logboek.Context(ctx).Default().LogProcess("Fetching origin").DoError(func() error {
		fetchOptions := true_git.FetchOptions{
			Unshallow: isShallow,
			RefSpecs:  map[string]string{"origin": "+refs/heads/*:refs/remotes/origin/*"},
		}

		if err := true_git.Fetch(ctx, repo.Path, fetchOptions); err != nil {
			return fmt.Errorf("fetch failed: %s", err)
		}

		return nil
	})
}

func (repo *Local) IsShallowClone() (bool, error) {
	return true_git.IsShallowClone(repo.Path)
}

func (repo *Local) CreateDetachedMergeCommit(ctx context.Context, fromCommit, toCommit string) (string, error) {
	return repo.createDetachedMergeCommit(ctx, repo.GitDir, repo.Path, repo.getRepoWorkTreeCacheDir(repo.getRepoID()), fromCommit, toCommit)
}

func (repo *Local) GetMergeCommitParents(_ context.Context, commit string) ([]string, error) {
	return repo.getMergeCommitParents(repo.GitDir, commit)
}

type LsTreeOptions struct {
	Commit        string
	UseHeadCommit bool
	Strict        bool
}

func (repo *Local) LsTree(ctx context.Context, pathMatcher path_matcher.PathMatcher, opts LsTreeOptions) (*ls_tree.Result, error) {
	repository, err := git.PlainOpenWithOptions(repo.Path, &git.PlainOpenOptions{EnableDotGitCommonDir: true})
	if err != nil {
		return nil, fmt.Errorf("cannot open repo %s: %s", repo.Path, err)
	}

	var commit string
	if opts.UseHeadCommit {
		commit = repo.headCommit
	} else if opts.Commit == "" {
		panic(fmt.Sprintf("no commit specified for LsTree procedure: specify Commit or HeadCommit"))
	} else {
		commit = opts.Commit
	}

	return ls_tree.LsTree(ctx, repository, commit, pathMatcher, opts.Strict)
}

func (repo *Local) Status(ctx context.Context, pathMatcher path_matcher.PathMatcher) (*status.Result, error) {
	repository, err := git.PlainOpenWithOptions(repo.Path, &git.PlainOpenOptions{EnableDotGitCommonDir: true})
	if err != nil {
		return nil, fmt.Errorf("cannot open repo %s: %s", repo.Path, err)
	}

	return status.Status(ctx, repository, repo.Path, pathMatcher)
}

func (repo *Local) IsEmpty(ctx context.Context) (bool, error) {
	return repo.isEmpty(ctx, repo.Path)
}

func (repo *Local) IsAncestor(_ context.Context, ancestorCommit, descendantCommit string) (bool, error) {
	return true_git.IsAncestor(ancestorCommit, descendantCommit, repo.GitDir)
}

func (repo *Local) RemoteOriginUrl(ctx context.Context) (string, error) {
	return repo.remoteOriginUrl(repo.Path)
}

func (repo *Local) HeadCommit(_ context.Context) (string, error) {
	return repo.headCommit, nil
}

func (repo *Local) CreatePatch(ctx context.Context, opts PatchOptions) (Patch, error) {
	return repo.createPatch(ctx, repo.Path, repo.GitDir, repo.getRepoID(), repo.getRepoWorkTreeCacheDir(repo.getRepoID()), opts)
}

func (repo *Local) CreateArchive(ctx context.Context, opts ArchiveOptions) (Archive, error) {
	return repo.createArchive(ctx, repo.Path, repo.GitDir, repo.getRepoID(), repo.getRepoWorkTreeCacheDir(repo.getRepoID()), opts)
}

func (repo *Local) Checksum(ctx context.Context, opts ChecksumOptions) (checksum Checksum, err error) {
	logboek.Context(ctx).Debug().LogProcess("Calculating checksum").Do(func() {
		checksum, err = repo.checksumWithLsTree(ctx, repo.Path, repo.GitDir, repo.getRepoWorkTreeCacheDir(repo.getRepoID()), opts)
	})

	return checksum, err
}

func (repo *Local) CheckAndReadCommitSymlink(ctx context.Context, path string, commit string) (bool, []byte, error) {
	return repo.checkAndReadSymlink(ctx, repo.getRepoWorkTreeCacheDir(repo.getRepoID()), repo.Path, repo.GitDir, commit, path)
}

func (repo *Local) IsCommitExists(ctx context.Context, commit string) (bool, error) {
	return repo.isCommitExists(ctx, repo.Path, repo.GitDir, commit)
}

func (repo *Local) TagsList(ctx context.Context) ([]string, error) {
	return repo.tagsList(repo.Path)
}

func (repo *Local) RemoteBranchesList(ctx context.Context) ([]string, error) {
	return repo.remoteBranchesList(repo.Path)
}

func (repo *Local) getRepoID() string {
	absPath, err := filepath.Abs(repo.Path)
	if err != nil {
		panic(err) // stupid interface of filepath.Abs
	}

	fullPath := filepath.Clean(absPath)
	return util.Sha256Hash(fullPath)
}

func (repo *Local) getRepoWorkTreeCacheDir(repoID string) string {
	return filepath.Join(GetWorkTreeCacheDir(), "local", repoID)
}

func (repo *Local) WalkCommitFilesWithGlob(ctx context.Context, commit string, dir string, glob string) ([]string, error) {
	glob = filepath.ToSlash(glob)
	matchFunc := func(path string) (bool, error) {
		for _, glob := range []string{
			glob,
			pathPkg.Join(glob, "**", "*"),
		} {
			matched, err := doublestar.Match(glob, path)
			if err != nil {
				return false, err // TODO
			}

			if matched {
				return true, nil
			}
		}

		return false, nil
	}

	var result []string
	if err := repo.WalkCommitFiles(ctx, commit, dir, func(notResolvedPath string) error {
		matched, err := matchFunc(notResolvedPath)
		if err != nil {
			return err // TODO
		}

		if matched {
			result = append(result, notResolvedPath)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (repo *Local) WalkCommitFiles(ctx context.Context, commit string, dir string, fileFunc func(notResolvedPath string) error) error {
	exist, err := repo.IsCommitDirectoryExist(ctx, commit, dir)
	if err != nil {
		return err
	}

	if !exist {
		return nil
	}

	resolvedDir, err := repo.ResolveCommitFilePath(ctx, commit, dir)
	if err != nil {
		return fmt.Errorf("unable to resolve commit file path %s: %s", dir, err)
	}

	result, err := repo.LsTree(ctx, path_matcher.NewSimplePathMatcher(resolvedDir, []string{}, true), LsTreeOptions{
		Commit: commit,
		Strict: true,
	})
	if err != nil {
		return err
	}

	return result.Walk(func(lsTreeEntry *ls_tree.LsTreeEntry) error {
		notResolvedPath := strings.Replace(filepath.ToSlash(lsTreeEntry.FullFilepath), resolvedDir, dir, -1)

		if lsTreeEntry.Mode.IsMalformed() { // TODO broken symlinks here
			return fmt.Errorf("broken symlinks here")
		} else if lsTreeEntry.Mode == filemode.Symlink {
			isDir, err := repo.IsCommitDirectoryExist(ctx, commit, notResolvedPath)
			if err != nil {
				return fmt.Errorf("!!!!")
			}

			if isDir {
				err := repo.WalkCommitFiles(ctx, commit, notResolvedPath, fileFunc)
				if err != nil {
					return err
				}
			} else {
				if err := fileFunc(notResolvedPath); err != nil {
					return err
				}
			}
		} else {
			if err := fileFunc(notResolvedPath); err != nil {
				return err
			}
		}

		return nil
	})
}

func (repo *Local) GetCommitFilePathList(ctx context.Context, commit string, dir string) ([]string, error) {
	resolvedDir, err := repo.ResolveCommitFilePath(ctx, commit, dir)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve commit file path %s: %s", dir, err)
	}

	result, err := repo.LsTree(ctx, path_matcher.NewGitMappingPathMatcher(resolvedDir, nil, nil, true), LsTreeOptions{
		Commit: commit,
		Strict: true,
	})
	if err != nil {
		return nil, err
	}

	var res []string
	if err := result.Walk(func(lsTreeEntry *ls_tree.LsTreeEntry) error {
		if lsTreeEntry.Mode.IsMalformed() { // TODO broken symlinks here

		} else if lsTreeEntry.Mode == filemode.Symlink {
			symlinkRes, err := repo.GetCommitFilePathList(ctx, commit, filepath.ToSlash(lsTreeEntry.FullFilepath))
			if err != nil {
				return err
			}

			for _, path := range symlinkRes {
				res = append(res, strings.Replace(path, resolvedDir, dir, -1))
			}
		} else {
			res = append(res, strings.Replace(lsTreeEntry.FullFilepath, resolvedDir, dir, -1))
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

// ReadCommitFile resolves symlinks and returns commit tree entry content.
func (repo *Local) ReadCommitFile(ctx context.Context, commit, path string) ([]byte, error) {
	resolvedPath, err := repo.ResolveCommitFilePath(ctx, commit, path)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve commit file path %s: %s", path, err)
	}

	return repo.getCommitTreeEntryContent(ctx, commit, resolvedPath)
}

// IsCommitFileExist resolves symlinks and returns true if the resolved commit tree entry is Regular, Deprecated, or Executable.
func (repo *Local) IsCommitFileExist(ctx context.Context, commit, path string) (bool, error) {
	resolvedPath, err := repo.ResolveCommitFilePath(ctx, commit, path)
	if err != nil {
		if err == EntryNotFoundInRepoErr {
			return false, nil
		}

		return false, fmt.Errorf("unable to resolve commit file path %s: %s", path, err)
	}

	lsTreeEntry, err := repo.getCommitTreeEntry(ctx, commit, resolvedPath)
	if err != nil {
		return false, fmt.Errorf("unable to get commit tree entry %s: %s", resolvedPath, err)
	}

	return lsTreeEntry.Mode.IsFile(), nil
}

// IsCommitDirectoryExist resolves symlinks and returns true if the resolved commit tree entry is Dir or Submodule.
func (repo *Local) IsCommitDirectoryExist(ctx context.Context, commit, path string) (bool, error) {
	resolvedPath, err := repo.ResolveCommitFilePath(ctx, commit, path)
	if err != nil {
		if err == EntryNotFoundInRepoErr {
			return false, nil
		}

		return false, fmt.Errorf("unable to resolve commit file path %s: %s", path, err)
	}

	lsTreeEntry, err := repo.getCommitTreeEntry(ctx, commit, resolvedPath)
	if err != nil {
		return false, fmt.Errorf("unable to get commit tree entry %s: %s", resolvedPath, err)
	}

	switch lsTreeEntry.Mode {
	case filemode.Dir, filemode.Submodule:
		return true, nil
	default:
		return false, nil
	}
}

// CheckAndResolveCommitFilePath does ResolveCommitFilePath with an additional check for each resolved path.
func (repo *Local) CheckAndResolveCommitFilePath(ctx context.Context, commit, path string, checkFunc func(relPath string) error) (string, error) {
	resolvedPath, err := repo.resolveCommitFilePath(ctx, commit, path, 0, checkFunc)
	fmt.Printf("%q %q %q %q %q\n", "check and resolve", path, resolvedPath, commit, err)
	return resolvedPath, err
}

// ResolveCommitFilePath follows symbolic links and returns the resolved path if there is a corresponding tree entry in the repo.
func (repo *Local) ResolveCommitFilePath(ctx context.Context, commit, path string) (string, error) {
	resolvedPath, err := repo.resolveCommitFilePath(ctx, commit, path, 0, nil)
	fmt.Printf("%q %q %q %q %q\n", "resolve", path, resolvedPath, commit, err)
	return resolvedPath, err
}

func (repo *Local) resolveCommitFilePath(ctx context.Context, commit, path string, depth int, checkFunc func(resolvedPath string) error) (string, error) {
	if depth > 1000 {
		return "", TooManyLevelsOfSymbolicLinksErr
	}
	depth++

	// returns path if the corresponding tree entry is Regular, Deprecated, Executable, Dir, or Submodule.
	{
		lsTreeEntry, err := repo.getCommitTreeEntry(ctx, commit, path)
		if err != nil {
			return "", fmt.Errorf("unable to get commit tree entry %s: %s", path, err)
		}

		if !lsTreeEntry.Mode.IsMalformed() && !(lsTreeEntry.Mode == filemode.Symlink) {
			if checkFunc != nil {
				if err := checkFunc(path); err != nil {
					return "", err
				}
			}

			return path, nil
		}
	}

	pathParts := util.SplitFilepath(path)
	pathPartsLen := len(pathParts)

	var resolvedPath string
	for ind := 0; ind < pathPartsLen; ind++ {
		isLastPathPart := ind == pathPartsLen-1
		pathToResolve := pathPkg.Join(resolvedPath, pathParts[ind])

		lsTreeEntry, err := repo.getCommitTreeEntry(ctx, commit, pathToResolve)
		if err != nil {
			return "", fmt.Errorf("unable to get commit tree entry %s: %s", pathToResolve, err)
		}

		mode := lsTreeEntry.Mode
		switch {
		case mode.IsMalformed(): // broken symlink here
			fmt.Println("malformed")
			return "", EntryNotFoundInRepoErr
		case mode == filemode.Symlink:
			data, err := repo.getCommitTreeEntryContent(ctx, commit, pathToResolve)
			if err != nil {
				return "", fmt.Errorf("unable to get commit tree entry content %s: %s", pathToResolve, err)
			}

			link := string(data)
			if pathPkg.IsAbs(link) {
				fmt.Println("absolute")
				return "", EntryNotFoundInRepoErr
			}

			resolvedLink := pathPkg.Clean(pathPkg.Join(pathPkg.Dir(pathToResolve), link))
			if resolvedLink == ".." || strings.HasPrefix(resolvedLink, "../") {
				fmt.Println("outside directory")
				return "", EntryNotFoundInRepoErr
			}

			fmt.Println(commit, link, depth)
			resolvedTarget, err := repo.resolveCommitFilePath(ctx, commit, link, depth, checkFunc)
			if err != nil {
				return "", err
			}

			resolvedPath = resolvedTarget
		case mode.IsFile() && !isLastPathPart:
			fmt.Println("broken path")
			return "", EntryNotFoundInRepoErr
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

func (repo *Local) getCommitTreeEntryContent(ctx context.Context, commit, relPath string) ([]byte, error) {
	lsTreeResult, err := repo.LsTree(ctx, path_matcher.NewSimplePathMatcher(relPath, []string{}, false), LsTreeOptions{
		Commit: commit,
		Strict: true,
	})
	if err != nil {
		return nil, err
	}

	return lsTreeResult.LsTreeEntryContent(relPath)
}

func (repo *Local) getCommitTreeEntry(ctx context.Context, commit, path string) (*ls_tree.LsTreeEntry, error) {
	lsTreeResult, err := repo.LsTree(ctx, path_matcher.NewSimplePathMatcher(path, []string{}, false), LsTreeOptions{
		Commit: commit,
		Strict: true,
	})
	if err != nil {
		return nil, err
	}

	entry := lsTreeResult.LsTreeEntry(path)

	return entry, nil
}
