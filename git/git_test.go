package git

import (
	"bytes"
	"fmt"
	"github.com/eighty4/maestro/testutil"
	"github.com/eighty4/maestro/util"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	util.InitLoggingWithLevel("ERROR")
	m.Run()
}

func TestClone_DirExists(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(dir, "https://github.com/eighty4/sse")
	if update := <-c; update.Status != Cloning {
		t.Fatal()
	}
	if update := <-c; update.Status != Cloned {
		t.Fatal()
	}
}

func TestClone_DirDoesNotExist(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(path.Join(dir, "sse"), "https://github.com/eighty4/sse")
	if update := <-c; update.Status != Cloning {
		t.Fatal()
	}
	if update := <-c; update.Status != Cloned {
		t.Fatal()
	}
}

func TestClone_Fails(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c := Clone(dir, "https://yahoo.com")
	if update := <-c; update.Status != Cloning {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-c; update.Status != CloneFailed {
		t.Fatal()
	} else {
		assert.Equal(t, "exit status 128", update.Message)
	}
}

func TestPull(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, dir, 1)

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != Pulled && update.PulledCommits != 1 {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
		assert.Equal(t, 0, update.RepoState.LocalCommits)
	}
}

func TestPull_WithLocalCommits(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.CommitNewFile(t, dir, "file1")

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != Pulled && update.PulledCommits != 1 {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
		assert.Equal(t, 1, update.RepoState.LocalCommits)
	}
}

func TestPull_Fails_DirNotRepo(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != PullFailed {
		t.Fatal(fmt.Sprintf("%s (expected) != %s (actual)", PullFailed, update.Status))
	} else {
		assert.Equal(t, "exit status 128", update.Message)
	}
}

func TestPull_Fails_WithDetachedHead(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.Checkout(t, dir, "5692a1bb7f5796ec3c0237c8cb0a87212b36b91e")

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != DetachedHead {
		t.Fatal(fmt.Sprintf("%s (expected) != %s (actual)", DetachedHead, update.Status))
	} else {
		assert.Equal(t, "detached from a branch", update.Message)
	}
}

func TestPull_Fails_WithDivergentBranches(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, dir, 1)
	testutil.CommitNewFile(t, dir, "file1")

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != DivergentBranches {
		t.Fatal()
	} else {
		assert.Equal(t, "divergent branches (require a merge or rebase)", update.Message)
	}
}

func TestPull_Fails_WithMergeConflict_WhenMerging(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.ResetHard(t, dir, 1)
	testutil.OpenFileForOverwriting(t, dir, "README.md", func(f *os.File) {
		if _, err := f.WriteString("merge conflict"); err != nil {
			t.Fatal(err)
		}
	})

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != MergeConflict {
		t.Fatal(fmt.Sprintf("%s (expected) != %s (actual)", MergeConflict, update.Status))
	} else {
		assert.Equal(t, "merge conflict", update.Message)
	}
}

func TestPull_Fails_WithMergeConflict_WhenRebasing(t *testing.T) {
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

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != MergeConflict {
		t.Fatal(fmt.Sprintf("%s (expected) != %s (actual)", MergeConflict, update.Status))
	} else {
		assert.Equal(t, "merge conflict (don't worry, rebase was aborted)", update.Message)
	}

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

func TestPull_Fails_WithUnsetUpstream(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.InitRepo(t, dir)

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != UnsetUpstream {
		t.Fatal(fmt.Sprintf("%s (expected) != %s (actual)", UnsetUpstream, update.Status))
	} else {
		assert.Equal(t, "not tracking an upstream remote", update.Message)
	}
}

func TestPull_Fails_WithUnstagedChanges(t *testing.T) {
	t.Skip("this scenario requires parameterizing `git pull` with `--rebase`")
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.CloneRepo(t, dir, "https://github.com/eighty4/sse")
	testutil.OpenFileForWriting(t, dir, "README.md", func(f *os.File) {
		if _, err := f.WriteString("unstaged change"); err != nil {
			t.Fatal(err)
		}
	})

	p := Pull(dir)
	if update := <-p; update.Status != Pulling {
		t.Fatal()
	} else {
		assert.Equal(t, "", update.Message)
	}
	if update := <-p; update.Status != UnstagedChanges {
		t.Fatal(fmt.Sprintf("%s (expected) != %s (actual)", UnstagedChanges, update.Status))
	} else {
		assert.Equal(t, "unstaged changes", update.Message)
	}
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
