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
	if os.Getenv("MAESTRO_TEST_GIT") != "true" {
		t.Skip("skipping git integration tests")
	}
}

func skipOnCi(t *testing.T) {
	if os.Getenv("MAESTRO_CI") == "true" {
		t.Skip("skipping on ci test run")
	}
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

func assertNotRebasing(t *testing.T, dir string) {
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

func TestPull_WithUnstagedChanges(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.OpenFileForWriting(t, dir, "README.md", func(f *os.File) {
		if _, err := f.WriteString("unstaged change"); err != nil {
			t.Fatal(err)
		}
	})

	testPullChannel(t, Pull(dir), Pulled, "", &RepoState{LocalCommits: 0})
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

const pullNotPossibleToFastForward = `
fatal: Not possible to fast-forward, aborting.
`

func TestMakePullErrorUpdate_ForMergeConflict(t *testing.T) {
	u := makePullErrorUpdate(pullNotPossibleToFastForward[1:])
	assert.Equal(t, MergeConflict, u.Status)
	assert.Equal(t, "unable to pull without a merge or interactive rebase", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithMergeConflict(t *testing.T) {
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
	testutil.AddAndCommit(t, dir, "README.md")

	testPullChannel(t, Pull(dir), MergeConflict, "unable to pull without a merge or interactive rebase", nil)
	assertNotRebasing(t, dir)
}

const pullMergeConflict = `
error: Your local changes to the following files would be overwritten by merge:
	README.md
Please commit your changes or stash them before you merge.
Aborting`

func TestMakePullErrorUpdate_ForOverwritesLocalChanges(t *testing.T) {
	u := makePullErrorUpdate(pullMergeConflict[1:])
	assert.Equal(t, OverwritesLocalChanges, u.Status)
	assert.Equal(t, "local changes would be overwritten", u.Message)
	assert.Nil(t, u.RepoState)
}

func TestPull_Fails_WithOverwritesLocalChanges(t *testing.T) {
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

	testPullChannel(t, Pull(dir), OverwritesLocalChanges, "local changes would be overwritten", nil)
	assertNotRebasing(t, dir)
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
	t.Skip("`git pull` fails with `unable to fork` when running all tests in git_test.go, passes individually")
	skipOnCi(t)
	gitIntegrationTest(t)
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
	t.Skip("not really testable")
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

func TestStashList(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.MkFile(t, dir, "stashed_file")
	testutil.GitAdd(t, dir, "stashed_file")
	gitStashCmd := exec.Command("git", "stash")
	gitStashCmd.Dir = dir
	if err := gitStashCmd.Run(); err != nil {
		t.Fatal(err)
	}

	stashes, err := StashList(dir)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(stashes))
	assert.Equal(t, "stash@{0}", stashes[0].Name)
	assert.Equal(t, "main", stashes[0].OnBranch)
	assert.Equal(t, "adding pkg.go.dev badge", stashes[0].Description)
	assert.Equal(t, "7f04bd8", stashes[0].OnCommitHash)
	gitRevParseCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	gitRevParseCmd.Dir = dir
	var stdoutBuf bytes.Buffer
	gitRevParseCmd.Stdout = &stdoutBuf
	if err := gitRevParseCmd.Run(); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, strings.TrimSpace(stdoutBuf.String()), stashes[0].OnCommitHash)
}

func TestStashList_EmptyStash(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")

	stashes, err := StashList(dir)
	assert.Nil(t, err)
	assert.Nil(t, stashes)
}

func TestStashList_Errors_NotRepository(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	stashes, err := StashList(dir)
	assert.Nil(t, stashes)
	assert.NotNil(t, err)
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
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.CommitNewFile(t, dir, "file1")
	testutil.CommitNewFile(t, dir, "file2")

	s, err := Status(dir)
	assert.Nil(t, err)
	assert.Equal(t, 2, s.LocalCommits)
	assert.Equal(t, 0, s.StagedChanges)
	assert.Equal(t, 0, s.UnstagedChanges)
	assert.Equal(t, 0, s.UntrackedFiles)
}

func TestStatus_WithStagedChanges(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.OpenFileForOverwriting(t, dir, "LICENSE", func(f *os.File) {
		_, _ = f.WriteString("license")
	})
	testutil.OpenFileForOverwriting(t, dir, "README.md", func(f *os.File) {
		_, _ = f.WriteString("readme")
	})
	testutil.GitAdd(t, dir, "LICENSE")
	testutil.GitAdd(t, dir, "README.md")

	s, err := Status(dir)
	assert.Nil(t, err)
	assert.Equal(t, 0, s.LocalCommits)
	assert.Equal(t, 2, s.StagedChanges)
	assert.Equal(t, 0, s.UnstagedChanges)
	assert.Equal(t, 0, s.UntrackedFiles)
}

func TestStatus_WithUnstagedChanges(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.OpenFileForOverwriting(t, dir, "LICENSE", func(f *os.File) {
		_, _ = f.WriteString("license")
	})
	testutil.OpenFileForOverwriting(t, dir, "README.md", func(f *os.File) {
		_, _ = f.WriteString("readme")
	})

	s, err := Status(dir)
	assert.Nil(t, err)
	assert.Equal(t, 0, s.LocalCommits)
	assert.Equal(t, 0, s.StagedChanges)
	assert.Equal(t, 2, s.UnstagedChanges)
	assert.Equal(t, 0, s.UntrackedFiles)
}

func TestStatus_WithUntrackedFiles(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.MkFile(t, dir, "new_file")
	testutil.MkFile(t, dir, "another_new_file")

	s, err := Status(dir)
	assert.Nil(t, err)
	assert.Equal(t, 0, s.LocalCommits)
	assert.Equal(t, 0, s.StagedChanges)
	assert.Equal(t, 0, s.UnstagedChanges)
	assert.Equal(t, 2, s.UntrackedFiles)
}

func TestStatus_WithAllStateTypes(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.CommitNewFile(t, dir, "file1")
	testutil.OpenFileForOverwriting(t, dir, "LICENSE", func(f *os.File) {
		_, _ = f.WriteString("license")
	})
	testutil.OpenFileForOverwriting(t, dir, "README.md", func(f *os.File) {
		_, _ = f.WriteString("readme")
	})
	testutil.GitAdd(t, dir, "README.md")
	testutil.MkFile(t, dir, "new_file")

	s, err := Status(dir)
	assert.Nil(t, err)
	assert.Equal(t, 1, s.LocalCommits)
	assert.Equal(t, 1, s.StagedChanges)
	assert.Equal(t, 1, s.UnstagedChanges)
	assert.Equal(t, 1, s.UntrackedFiles)
}

func TestParsePullCommitRange(t *testing.T) {
	const gitPullStdout = `Updating 700148d..7f04bd8
Fast-forward
 README.md | 3 +++
 sse.go    | 1 +
 2 files changed, 4 insertions(+)
`
	from, to, err := parsePulledCommitRange(gitPullStdout)
	assert.Nil(t, err)
	assert.Equal(t, "700148d", from)
	assert.Equal(t, "7f04bd8", to)
}

func TestParsePullCommitRange_BadInput(t *testing.T) {
	_, _, err := parsePulledCommitRange("2\n")
	assert.Equal(t, "failed to find commit hashes in git pull stdout", err.Error())
}

func TestParsePullCommitRange_GoodInput(t *testing.T) {
	input := "Updating 0115386a..3456ecd9\nFast-forward\n"
	from, to, err := parsePulledCommitRange(input)
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

func TestGetPulledCommitCount(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")

	stdout := `Updating ea0888b..7f04bd8
Fast-forward
 README.md | 5 ++++-
 sse.go    | 1 +
 2 files changed, 5 insertions(+), 1 deletion(-)`

	count, err := getPulledCommitCount(dir, stdout)
	assert.Nil(t, err)
	assert.Equal(t, count, 3)
}

func TestGetPulledCommitCount_WhenAlreadyUpToDate(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	count, err := getPulledCommitCount(dir, "Already up to date.")
	assert.Nil(t, err)
	assert.Equal(t, count, 0)
}

func TestParsePulledCommitCount(t *testing.T) {
	stdout := `Updating ea0888b..7f04bd8
Fast-forward
 README.md | 5 ++++-
 sse.go    | 1 +
 2 files changed, 5 insertions(+), 1 deletion(-)`

	from, to, err := parsePulledCommitRange(stdout)
	assert.Nil(t, err)
	assert.Equal(t, from, "ea0888b")
	assert.Equal(t, to, "7f04bd8")
}

func TestRevListCommitCount(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")

	count, err := RevListCommitCount(dir, "9a6992f988bee6e47540e53e434ad07911db3a30", "700148df1ec546c06ce1a54bc472fee3085a2842")
	assert.Nil(t, err)
	assert.Equal(t, count, 2)
}
