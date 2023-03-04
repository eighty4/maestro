package git

import (
	"github.com/eighty4/maestro/testutil"
	"github.com/stretchr/testify/assert"
	"path"
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
	repoDir := path.Join(dir, "repo")
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

	repos := []*Repository{NewRepository("sse", path.Join(dir, "sse"), "https://github.com/eighty4/sse")}
	work := NewWorkspace(dir, repos, 0)
	c := work.Sync()
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

func TestWorkspace_Sync_PullsRepo(t *testing.T) {
	gitIntegrationTest(t)
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, path.Join(dir, "sse"), "https://github.com/eighty4/sse")

	work := NewWorkspace(dir, []*Repository{}, 1)
	c := work.Sync()
	update, ok := <-c
	assert.True(t, ok)
	assert.Equal(t, SyncSuccess, update.Status)
	assert.Equal(t, PullSync, update.Op)
	assert.Equal(t, "sse", update.Repo)
	assert.Equal(t, "", update.Message)
	update, ok = <-c
	assert.False(t, ok)
	assert.Nil(t, update)
}
