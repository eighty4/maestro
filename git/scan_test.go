package git

import (
	"github.com/eighty4/maestro/testutil"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestScanForRepositories_RepositoriesInRootDir(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.MkDirAndInitRepo(t, filepath.Join(dir, "dir1"))
	testutil.MkDirAndInitRepo(t, filepath.Join(dir, "dir2"))
	testutil.MkDir(t, filepath.Join(dir, "dir3"))
	testutil.MkDir(t, filepath.Join(dir, "dir4"))
	testutil.MkFile(t, dir, "file1")
	testutil.MkFile(t, dir, "file2")

	repositories := ScanForRepositories(dir, 0)
	assert.Len(t, repositories, 2)
}

func TestScanForRepositories_NestedRepositories_BeyondScanDepth(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.MkDir(t, filepath.Join(dir, "dir1"))
	testutil.MkDir(t, filepath.Join(dir, "dir2"))
	testutil.MkDirAndInitRepo(t, filepath.Join(dir, "dir1", "subdir1"))
	testutil.MkDirAndInitRepo(t, filepath.Join(dir, "dir1", "subdir2"))
	testutil.MkFile(t, dir, "file1")
	testutil.MkFile(t, dir, "file2")

	repositories := ScanForRepositories(dir, 0)
	assert.Len(t, repositories, 0)
}

func TestScanForRepositories_NestedRepositories_WithinScanDepth(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.MkDir(t, filepath.Join(dir, "dir1"))
	testutil.MkDir(t, filepath.Join(dir, "dir2"))
	testutil.MkDirAndInitRepo(t, filepath.Join(dir, "dir1", "subdir1"))
	testutil.MkDirAndInitRepo(t, filepath.Join(dir, "dir1", "subdir2"))
	testutil.MkFile(t, dir, "file1")
	testutil.MkFile(t, dir, "file2")

	repositories := ScanForRepositories(dir, 1)
	assert.Len(t, repositories, 2)
}

func TestScanForRepositories_NoSubdirectories(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.MkFile(t, dir, "file1")
	testutil.MkFile(t, dir, "file2")

	repositories := ScanForRepositories(dir, 0)
	assert.Len(t, repositories, 0)
}

func TestScanForRepositories_NoRepositories(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.MkDir(t, filepath.Join(dir, "dir1"))
	testutil.MkDir(t, filepath.Join(dir, "dir2"))
	testutil.MkFile(t, dir, "file1")
	testutil.MkFile(t, dir, "file2")

	repositories := ScanForRepositories(dir, 0)
	assert.Len(t, repositories, 0)
}

func TestSubdirectories_GibberishDir(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	_, err := subdirectories(filepath.Join(dir, "asdf"))
	if err == nil {
		t.Fatal(err)
	}
}

func TestSubdirectories_WithDir(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.MkDir(t, filepath.Join(dir, "dir"))
	testutil.MkFile(t, dir, "file")

	r, err := subdirectories(dir)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, r, 1)
	assert.Equal(t, "dir", r[0])
}

func TestSubdirectories_NoDir(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.MkFile(t, dir, "file")

	r, err := subdirectories(dir)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, r, 0)
}

func TestIsGitRepoRootDir_RepoRoot(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.InitRepo(t, dir)

	assert.True(t, isTopLevelGitRepoDir(dir))
}

func TestIsGitRepoRootDir_RepoSubdir(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.InitRepo(t, dir)
	testutil.MkDir(t, filepath.Join(dir, "subdir"))

	assert.False(t, isTopLevelGitRepoDir(filepath.Join(dir, "subdir")))
}

func TestIsGitRepoRootDir_NotRepo(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	assert.False(t, isTopLevelGitRepoDir(dir))
}
