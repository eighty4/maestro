package git

import (
	"github.com/eighty4/maestro/testutil"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewWorkspace_WithoutRepoScan(t *testing.T) {
	repos := []*Repository{NewRepository("repo", "/work/repo", "https://github.com/eighty4/repo")}
	work := NewWorkspace("/work", repos, 0)
	assert.Equal(t, "/work", work.RootDir)
	assert.Equal(t, repos[0], work.Repositories["/work/repo"])
}

func TestNewWorkspace_WithRepoScan(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	repoDir := filepath.Join(dir, "repo")
	testutil.MkDirAndInitRepo(t, repoDir)

	var repos []*Repository
	work := NewWorkspace(dir, repos, 1)
	assert.Len(t, work.Repositories, 1)
	assert.Equal(t, "repo", work.Repositories[repoDir].Name)
	assert.Equal(t, repoDir, work.Repositories[repoDir].Dir)
	assert.Equal(t, "", work.Repositories[repoDir].Git.Url)
}

func TestWorkspace_Sync_ClonesRepo(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	repos := []*Repository{NewRepository("sse", filepath.Join(dir, "sse"), "https://github.com/eighty4/sse")}
	work := NewWorkspace(dir, repos, 0)
	c := work.Sync(nil)
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncSuccess, update.Status)
	assert.Equal(t, CloneSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "cloned from https://github.com/eighty4/sse", update.Message)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}

func TestWorkspace_Sync_ClonesRepo_Failure(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	repos := []*Repository{NewRepository("sse", filepath.Join(dir, "sse"), "https://github.com/asdgsadgasdgasgasdg")}
	work := NewWorkspace(dir, repos, 0)
	c := work.Sync(nil)
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncFailure, update.Status)
	assert.Equal(t, CloneSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "", update.Message)
	assert.Equal(t, "repository not found", update.Error)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}

func TestWorkspace_Sync_PullsRepo_WithPulledCommits(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	repoDir := filepath.Join(dir, "sse")
	testutil.CloneRepo(t, repoDir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, repoDir, 2)

	work := NewWorkspace(dir, []*Repository{}, 1)
	c := work.Sync(nil)
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncSuccess, update.Status)
	assert.Equal(t, PullSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "pulled 2 commits", update.Message)
	assert.Equal(t, "", update.Error)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}

func TestWorkspace_Sync_PullsRepo_WithLocalChanges(t *testing.T) {
	gitIntegrationTest(t)
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	repoDir := filepath.Join(dir, "sse")
	testutil.CloneRepo(t, repoDir, "https://github.com/eighty4/sse")
	testutil.CommitNewFile(t, repoDir, "file1")
	testutil.OpenFileForOverwriting(t, repoDir, "LICENSE", func(f *os.File) {
		_, _ = f.WriteString("license")
	})
	testutil.OpenFileForOverwriting(t, repoDir, "README.md", func(f *os.File) {
		_, _ = f.WriteString("readme")
	})
	testutil.GitAdd(t, repoDir, "README.md")
	testutil.MkFile(t, repoDir, "new_file")

	work := NewWorkspace(dir, []*Repository{}, 1)
	c := work.Sync(nil)
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncWarning, update.Status)
	assert.Equal(t, PullSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "1 local commit, 3 local changes", update.Message)
	assert.Equal(t, "", update.Error)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}

func TestWorkspace_Sync_PullsRepo_WithDetailedLocalChanges(t *testing.T) {
	gitIntegrationTest(t)
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	repoDir := filepath.Join(dir, "sse")
	testutil.CloneRepo(t, repoDir, "https://github.com/eighty4/sse")
	testutil.CommitNewFile(t, repoDir, "file1")
	testutil.OpenFileForOverwriting(t, repoDir, "LICENSE", func(f *os.File) {
		_, _ = f.WriteString("license")
	})
	testutil.OpenFileForOverwriting(t, repoDir, "README.md", func(f *os.File) {
		_, _ = f.WriteString("readme")
	})
	testutil.GitAdd(t, repoDir, "README.md")
	testutil.MkFile(t, repoDir, "new_file")

	work := NewWorkspace(dir, []*Repository{}, 1)
	c := work.Sync(&SyncOptions{DetailLocalChanges: true})
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncWarning, update.Status)
	assert.Equal(t, PullSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "1 local commit, 3 local changes (1 staged, 1 not staged, 1 untracked)", update.Message)
	assert.Equal(t, "", update.Error)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}

func TestWorkspace_Sync_PullsRepo_WithStashedChanges(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	repoDir := filepath.Join(dir, "sse")
	testutil.CloneRepo(t, repoDir, "https://github.com/eighty4/sse")
	testutil.MkFile(t, repoDir, "stashed_file")
	testutil.GitAdd(t, repoDir, "stashed_file")
	testutil.GitStash(t, repoDir)

	work := NewWorkspace(dir, []*Repository{NewRepository("sse", repoDir, "https://github.com/asdgsadgasdgasgasdg")}, 1)
	c := work.Sync(nil)
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncWarning, update.Status)
	assert.Equal(t, PullSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "1 stashed change", update.Message)
	assert.Equal(t, "", update.Error)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}

func TestWorkspace_Sync_PullsRepo_Failure(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	repoDir := filepath.Join(dir, "sse")
	testutil.MkDir(t, repoDir)
	testutil.InitRepo(t, repoDir)

	repos := []*Repository{NewRepository("sse", repoDir, "https://github.com/eighty4/sse")}
	work := NewWorkspace(dir, repos, 0)
	c := work.Sync(nil)
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncFailure, update.Status)
	assert.Equal(t, PullSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "", update.Message)
	assert.Equal(t, "not tracking an upstream remote", update.Error)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}
