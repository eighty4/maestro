package git

import (
	"github.com/eighty4/maestro/testutil"
	"github.com/stretchr/testify/assert"
	"path"
	"testing"
)

func TestNewWorkspace_WithoutRepoScan(t *testing.T) {
	repos := []*Repository{{"repo", "/work/repo", "https://github.com/eighty4/repo"}}
	work := NewWorkspace("/work", repos, 0)
	assert.Equal(t, "/work", work.RootDir)
	assert.Equal(t, repos[0], work.Repositories["/work/repo"])
}

func TestNewWorkspace_WithRepoScan(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	repoDir := path.Join(dir, "repo")
	testutil.MkDirAndInitGitRepo(t, repoDir)

	var repos []*Repository
	work := NewWorkspace(dir, repos, 1)
	assert.Len(t, work.Repositories, 1)
	assert.Equal(t, "repo", work.Repositories[repoDir].Name)
	assert.Equal(t, repoDir, work.Repositories[repoDir].Dir)
	assert.Equal(t, "", work.Repositories[repoDir].Url)
}
