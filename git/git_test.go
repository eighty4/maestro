package git

import (
	"bytes"
	"github.com/eighty4/maestro/testutil"
	"github.com/eighty4/maestro/util"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	util.InitLoggingWithLevel("ERROR")
	m.Run()
}

func gitIntegrationTest(t *testing.T) {
	//if os.Getenv("MAESTRO_GIT_TESTING") != "true" {
	//	t.Skip("set MAESTRO_GIT_TESTING=true to run integration tests")
	//}
}

func testCloneChannel(t *testing.T, c <-chan *CloneUpdate, expStatus CloneStatus, expMessage string) {
	u, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, Cloning, u.Status)
	assert.Equal(t, "", u.Message)
	u, ok = <-c
	assert.True(t, ok)
	assert.Equal(t, expStatus, u.Status)
	assert.Equal(t, expMessage, u.Message)
	u, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, u)
}

func TestClone_IntoExistingDir(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(dir, "https://github.com/eighty4/sse")
	testCloneChannel(t, c, Cloned, "")
}

func TestClone_IntoNewDir(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(path.Join(dir, "sse"), "https://github.com/eighty4/sse")
	testCloneChannel(t, c, Cloned, "")
}

const cloneAuthRequired = `
Cloning into 'asdf'...
fatal: could not read Username for 'https://github.com': terminal prompts disabled`

func TestMakeCloneErrorUpdate_ForAuthRequired(t *testing.T) {
	u := makeCloneErrorUpdate(cloneAuthRequired[1:])
	assert.Equal(t, AuthRequired, u.Status)
	assert.Equal(t, "authentication required", u.Message)
}

func TestClone_Fails_WithAuthFailed(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(dir, "https://github.com/eighty4/asdf")
	testCloneChannel(t, c, AuthRequired, "authentication required")
}

const cloneBadRedirect = `
fatal: unable to update url base from redirection:
  asked for: https://yahoo.com/info/refs?service=git-upload-pack
   redirect: https://www.yahoo.com/`

func TestMakeCloneErrorUpdate_ForBadRedirect(t *testing.T) {
	u := makeCloneErrorUpdate(cloneBadRedirect[1:])
	assert.Equal(t, BadRedirect, u.Status)
	assert.Equal(t, "following http redirect did not connect to a git repository", u.Message)
}

func TestClone_Fails_WithBadRedirect(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(dir, "https://yahoo.com")
	testCloneChannel(t, c, BadRedirect, "following http redirect did not connect to a git repository")
}

const cloneNotFound = `
remote: Not Found
fatal: repository 'https://github.com/asdgsadgasdgasgasdg/' not found`

func TestMakeCloneErrorUpdate_ForRepoNotFound(t *testing.T) {
	u := makeCloneErrorUpdate(cloneNotFound[1:])
	assert.Equal(t, CloneRepoNotFound, u.Status)
	assert.Equal(t, "repository not found", u.Message)
}

func TestClone_Fails_WithNotFound(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(dir, "https://github.com/asdgsadgasdgasgasdg")
	testCloneChannel(t, c, CloneRepoNotFound, "repository not found")
}

func testPullChannel(t *testing.T, p <-chan *PullUpdate, expStatus PullStatus, expMessage string, expRepoState *RepoState) {
	u, ok := <-p
	assert.True(t, ok)
	assert.Equal(t, Pulling, u.Status)
	assert.Equal(t, "", u.Message)
	u, ok = <-p
	assert.True(t, ok)
	assert.Equal(t, expStatus, u.Status)
	assert.Equal(t, expMessage, u.Message)
	if expRepoState == nil {
		assert.Nil(t, u.RepoState)
	} else {
		assert.Equal(t, expRepoState.LocalCommits, u.RepoState.LocalCommits)
	}
	u, ok = <-p
	assert.False(t, ok)
	assert.Nil(t, u)
}

func TestPull(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, dir, 1)

	testPullChannel(t, Pull(dir), Pulled, "", &RepoState{LocalCommits: 0})
}

func TestPull_WithLocalCommits(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.CommitNewFile(t, dir, "file1")

	testPullChannel(t, Pull(dir), Pulled, "", &RepoState{LocalCommits: 1})
}

func TestPull_Fails_DirNotRepo(t *testing.T) {
	gitIntegrationTest(t)
	_ = `
fatal: not a git repository (or any of the parent directories): .git`

	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	testPullChannel(t, Pull(dir), NotRepository, "not a repository", nil)
}

const pullDetachedHead = `
You are not currently on a branch.
Please specify which branch you want to merge with.
See git-pull(1) for details.

    git pull <remote> <branch>`

func TestMakePullErrorUpdate_ForDetachedHead(t *testing.T) {
	u := makePullErrorUpdate(pullDetachedHead[1:])
	assert.Equal(t, DetachedHead, u.Status)
	assert.Equal(t, "detached from a branch", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithDetachedHead(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.Checkout(t, dir, "5692a1bb7f5796ec3c0237c8cb0a87212b36b91e")

	testPullChannel(t, Pull(dir), DetachedHead, "detached from a branch", nil)
}

const pullDivergentBranches = `
hint: You have divergent branches and need to specify how to reconcile them.
hint: You can do so by running one of the following commands sometime before
hint: your next pull:
hint: 
hint:   git config pull.rebase false  # merge
hint:   git config pull.rebase true   # rebase
hint:   git config pull.ff only       # fast-forward only
hint: 
hint: You can replace "git config" with "git config --global" to set a default
hint: preference for all repositories. You can also pass --rebase, --no-rebase,
hint: or --ff-only on the command line to override the configured default per
hint: invocation.
fatal: Need to specify how to reconcile divergent branches.`

func TestMakePullErrorUpdate_ForDivergentBranches(t *testing.T) {
	u := makePullErrorUpdate(pullDivergentBranches[1:])
	assert.Equal(t, DivergentBranches, u.Status)
	assert.Equal(t, "divergent branches (require a merge or rebase)", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithDivergentBranches(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, dir, 1)
	testutil.CommitNewFile(t, dir, "file1")

	testPullChannel(t, Pull(dir), DivergentBranches, "divergent branches (require a merge or rebase)", nil)
}

const pullMergeConflict = `
error: Your local changes to the following files would be overwritten by merge:
	README.md
Please commit your changes or stash them before you merge.
Aborting`

func TestMakePullErrorUpdate_ForMergeConflict(t *testing.T) {
	u := makePullErrorUpdate(pullMergeConflict[1:])
	assert.Equal(t, MergeConflict, u.Status)
	assert.Equal(t, "merge conflict", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithMergeConflict_WhenMerging(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, dir, 1)
	testutil.OpenFileForOverwriting(t, dir, "README.md", func(f *os.File) {
		if _, err := f.WriteString("merge conflict"); err != nil {
			t.Fatal(err)
		}
	})

	testPullChannel(t, Pull(dir), MergeConflict, "merge conflict", nil)
}

const pullRebaseConflict = `
error: could not apply f4d2207... "README.md"
hint: Resolve all conflicts manually, mark them as resolved with
hint: "git add/rm <conflicted_files>", then run "git rebase --continue".
hint: You can instead skip this commit: run "git rebase --skip".
hint: To abort and get back to the state before "git rebase", run "git rebase --abort".`

func TestMakePullErrorUpdate_ForRebaseConflict(t *testing.T) {
	u := makePullErrorUpdate(pullRebaseConflict[1:])
	assert.Equal(t, MergeConflict, u.Status)
	assert.Equal(t, "merge conflict (don't worry, rebase was aborted)", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithMergeConflict_WhenRebasing(t *testing.T) {
	gitIntegrationTest(t)
	t.Skip("this scenario requires parameterizing `git pull` with `--rebase`")
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, dir, 1)
	testutil.OpenFileForOverwriting(t, dir, "README.md", func(f *os.File) {
		if _, err := f.WriteString("merge conflict"); err != nil {
			t.Fatal(err)
		}
	})
	testutil.AddAndCommit(t, dir, "README.md")

	testPullChannel(t, Pull(dir), MergeConflict, "merge conflict (don't worry, rebase was aborted)", nil)

	gitStatusCmd := exec.Command("git", "status")
	gitStatusCmd.Dir = dir
	var stdout bytes.Buffer
	gitStatusCmd.Stdout = &stdout
	if err := gitStatusCmd.Run(); err != nil {
		t.Fatal(err)
	} else {
		assert.False(t, strings.Contains(stdout.String(), "You are currently rebasing branch"))
	}
}

const pullConnectionFailure = `
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.`

func TestMakePullErrorUpdate_ForConnectionFailure(t *testing.T) {
	u := makePullErrorUpdate(pullConnectionFailure[1:])
	assert.Equal(t, ConnectionFailure, u.Status)
	assert.Equal(t, "connection failure with remote repository", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithConnectionFailure_WithoutFailureReason(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.SetGitRemoteOriginUrl(t, dir, "git@github.com:eighty4/sse")

	ogGscValue := os.Getenv("GIT_SSH_COMMAND")
	if err := os.Setenv("GIT_SSH_COMMAND", "false"); err != nil {
		t.Fatal(err.Error())
	}
	testPullChannel(t, Pull(dir), ConnectionFailure, "connection failure with remote repository", nil)
	_ = os.Setenv("GIT_SSH_COMMAND", ogGscValue)
}

const pullConnectionFailureWithReason = `
exit 1: line 0: exit: too many arguments
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.`

func TestMakePullErrorUpdate_ForConnectionFailureWithReason(t *testing.T) {
	u := makePullErrorUpdate(pullConnectionFailureWithReason[1:])
	assert.Equal(t, ConnectionFailure, u.Status)
	assert.Equal(t, "\"exit 1: line 0: exit: too many arguments\"", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithConnectionFailure_WithFailureReason(t *testing.T) {
	gitIntegrationTest(t)
	if runtime.GOOS != "darwin" {
		t.Skip("output does not match on linux")
	}
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.SetGitRemoteOriginUrl(t, dir, "git@github.com:aaaaaaaaaaaaaaaaaaaaaaaaaaa/aaaaaaaaaaaaaaaaaaaaaaaaaaa")

	ogGscValue := os.Getenv("GIT_SSH_COMMAND")
	if err := os.Setenv("GIT_SSH_COMMAND", "sleep foo"); err != nil {
		t.Fatal(err.Error())
	}
	testPullChannel(t, Pull(dir), ConnectionFailure, `"usage: sleep seconds"`, nil)
	_ = os.Setenv("GIT_SSH_COMMAND", ogGscValue)
}

const pullGitHubNotFound = `
ERROR: Repository not found.
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.`

func TestMakePullErrorUpdate_ForGitHubNotFound(t *testing.T) {
	u := makePullErrorUpdate(pullGitHubNotFound[1:])
	assert.Equal(t, PullRepoNotFound, u.Status)
	assert.Equal(t, "repository not found", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithGitHubRepoNotFoundConnectionError_MappedToRepositoryNotFound(t *testing.T) {
	gitIntegrationTest(t)
	t.Skip("unable to run on CI because it requires SSH auth to GitHub")
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.SetGitRemoteOriginUrl(t, dir, "git@github.com:eighty4/aaaaaaaaaaaaaaaaaaaaaaaaaaa")

	testPullChannel(t, Pull(dir), PullRepoNotFound, "repository not found", nil)
}

const pullCouldNotResolveHost = `
fatal: unable to access 'https://github.com/eighty4/sse/': Could not resolve host: github.com`

func TestMakePullErrorUpdate_ForCouldNotResolveHost(t *testing.T) {
	u := makePullErrorUpdate(pullCouldNotResolveHost[1:])
	assert.Equal(t, CouldNotResolveHost, u.Status)
	assert.Equal(t, "could not resolve host", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithCouldNotResolveHost(t *testing.T) {
	gitIntegrationTest(t)
	t.Skip("must disconnect from WiFi/ethernet between clone and pull or use namespace to deny network")
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://gibhub.com/eighty4/sse")

	testPullChannel(t, Pull(dir), CouldNotResolveHost, "could not resolve host", nil)
}

const pullRemoteBranchNotFound = `
Your configuration specifies to merge with the ref 'refs/heads/macrame'
from the remote, but no such ref was fetched.`

func TestMakePullErrorUpdate_ForRemoteBranchNotFound(t *testing.T) {
	u := makePullErrorUpdate(pullRemoteBranchNotFound[1:])
	assert.Equal(t, RemoteBranchNotFound, u.Status)
	assert.Equal(t, "tracking branch not found on remote", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithRemoteBranchNotFound(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")

	gitCommitCmd := exec.Command("git", "config", "branch.main.merge", "macrame")
	gitCommitCmd.Dir = dir
	var gitCommitCmdStderr bytes.Buffer
	gitCommitCmd.Stderr = &gitCommitCmdStderr
	if err := gitCommitCmd.Run(); err != nil {
		t.Fatal(gitCommitCmdStderr.String())
	}

	testPullChannel(t, Pull(dir), RemoteBranchNotFound, "tracking branch not found on remote", nil)
}

const pullUnsetUpstream = `
Initialized empty Git repository in /private/var/folders/r6/0hg_xym96qbgc8m049hxjrxr0000gn/T/maestro-test2377015730/.git/
There is no tracking information for the current branch.
Please specify which branch you want to merge with.
See git-pull(1) for details.

    git pull <remote> <branch>

If you wish to set tracking information for this branch you can do so with:

    git branch --set-upstream-to=<remote>/<branch> main`

func TestMakePullErrorUpdate_ForUnsetUpstream(t *testing.T) {
	u := makePullErrorUpdate(pullUnsetUpstream[1:])
	assert.Equal(t, UnsetUpstream, u.Status)
	assert.Equal(t, "not tracking an upstream remote", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithUnsetUpstream(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.InitRepo(t, dir)

	testPullChannel(t, Pull(dir), UnsetUpstream, "not tracking an upstream remote", nil)
}

const pullUnstagedChanges = `
error: cannot pull with rebase: You have unstaged changes.
error: please commit or stash them.`

func TestMakePullErrorUpdate_ForUnstagedChanges(t *testing.T) {
	u := makePullErrorUpdate(pullUnstagedChanges[1:])
	assert.Equal(t, UnstagedChanges, u.Status)
	assert.Equal(t, "unstaged changes", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithUnstagedChanges(t *testing.T) {
	gitIntegrationTest(t)
	t.Skip("this scenario requires parameterizing `git pull` with `--rebase`")
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.OpenFileForWriting(t, dir, "README.md", func(f *os.File) {
		if _, err := f.WriteString("unstaged change"); err != nil {
			t.Fatal(err)
		}
	})

	testPullChannel(t, Pull(dir), UnstagedChanges, "unstaged changes", nil)
}

const pullRepositoryNotFound = `
fatal: repository 'https://eighty4.io/' not found`

func TestMakePullErrorUpdate_ForRepositoryNotFound(t *testing.T) {
	u := makePullErrorUpdate(pullRepositoryNotFound[1:])
	assert.Equal(t, PullRepoNotFound, u.Status)
	assert.Equal(t, "repository not found", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithRepositoryNotFound(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.SetGitRemoteOriginUrl(t, dir, "https://eighty4.io")

	testPullChannel(t, Pull(dir), PullRepoNotFound, "repository not found", nil)
}

func TestRevParseShowTopLevel_NotRepo(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	result, err := RevParseShowTopLevel(dir)
	assert.NotNil(t, err)
	assert.Empty(t, result)
}

func TestRevParseShowTopLevel_TopLevelDir(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.InitRepo(t, dir)

	result, err := RevParseShowTopLevel(dir)
	assert.Nil(t, err)
	assert.Equal(t, dir, result)
}

func TestRevParseShowTopLevel_Subdir(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.InitRepo(t, dir)
	subdir := path.Join(dir, "foo")
	testutil.MkDir(t, subdir)

	result, err := RevParseShowTopLevel(subdir)
	assert.Nil(t, err)
	assert.Equal(t, dir, result)
}

func TestStatus_NoLocalCommits(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.InitRepo(t, dir)

	s, err := Status(dir)
	assert.Nil(t, err)
	assert.Equal(t, 0, s.LocalCommits)
}

func TestStatus_WithLocalCommits(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.CommitNewFile(t, dir, "file1")
	testutil.CommitNewFile(t, dir, "file2")

	s, err := Status(dir)
	assert.Nil(t, err)
	assert.Equal(t, 2, s.LocalCommits)
}

func TestParsePullCommitRange(t *testing.T) {
	const gitPullStdout = `Updating 700148d..7f04bd8
Fast-forward
 README.md | 3 +++
 sse.go    | 1 +
 2 files changed, 4 insertions(+)
`
	from, to, err := parsePullCommitRange(gitPullStdout)
	assert.Nil(t, err)
	assert.Equal(t, "700148d", from)
	assert.Equal(t, "7f04bd8", to)
}

func TestParsePullCommitRange_BadInput(t *testing.T) {
	_, _, err := parsePullCommitRange("2\n")
	assert.Equal(t, "failed to find commit hashes in git pull stdout", err.Error())
}

func TestParsePullCommitRange_GoodInput(t *testing.T) {
	input := "Updating 0115386a..3456ecd9\nFast-forward\n"
	from, to, err := parsePullCommitRange(input)
	assert.Nil(t, err)
	assert.Equal(t, "0115386a", from)
	assert.Equal(t, "3456ecd9", to)
}

func TestParseRevListCommitCount(t *testing.T) {
	commitCount, err := parseRevListCommitCount("2\n")
	assert.Nil(t, err)
	assert.Equal(t, 2, commitCount)
}

func TestParseRevListCommitCount_BadInput(t *testing.T) {
	_, err := parseRevListCommitCount("two\n")
	assert.Equal(t, "strconv.Atoi: parsing \"two\": invalid syntax", err.Error())
}

func TestParseRevParseAbsolutePath_TrimsInput(t *testing.T) {
	assert.Equal(t, "woo", parseRevParseAbsolutePath(" woo \n"))
}
