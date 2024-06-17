package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func MkDirAndInitRepo(t *testing.T, dir string) {
	MkDir(t, dir)
	InitRepo(t, dir)
}

func InitRepo(t *testing.T, dir string) {
	gitInit := exec.Command("git", "init")
	gitInit.Dir = dir
	gitInit.Stdout = os.Stdout
	gitInit.Stderr = os.Stderr
	if err := gitInit.Run(); err != nil {
		t.Fatal(err)
	}
}

func CloneRepo(t *testing.T, dir string, url string) {
	cloneCmd := exec.Command("git", "clone", url, dir)
	var stderr bytes.Buffer
	cloneCmd.Stderr = &stderr
	err := cloneCmd.Run()
	if err != nil {
		t.Fatal(stderr.String())
	} else if cloneCmd.ProcessState.ExitCode() != 0 {
		t.Fatal("git clone error")
	}
}

func Checkout(t *testing.T, dir string, branchOrCommitHash string) {
	checkoutCmd := exec.Command("git", "checkout", branchOrCommitHash)
	checkoutCmd.Dir = dir
	err := checkoutCmd.Run()
	if err != nil {
		t.Fatal(err)
	} else if checkoutCmd.ProcessState.ExitCode() != 0 {
		t.Fatal("git checkout error")
	}
}

func ResetHard(t *testing.T, dir string, commits uint8) {
	resetCmd := exec.Command("git", "reset", "--hard", fmt.Sprintf("HEAD~%d", commits))
	resetCmd.Dir = dir
	err := resetCmd.Run()
	if err != nil {
		t.Fatal(err)
	} else if resetCmd.ProcessState.ExitCode() != 0 {
		t.Fatal("git reset error")
	}
}

func GitAdd(t *testing.T, dir string, name string) {
	gitAddCmd := exec.Command("git", "add", name)
	gitAddCmd.Dir = dir
	if err := gitAddCmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func AddAndCommit(t *testing.T, dir string, name string) {
	GitAdd(t, dir, name)
	commit(t, dir, name, false)
}

func AddAndAmendCommit(t *testing.T, dir string, name string) {
	GitAdd(t, dir, name)
	commit(t, dir, name, true)
}

func commit(t *testing.T, dir string, name string, amend bool) {
	var args []string
	args = append(args, "commit", "-m", fmt.Sprintf(`"%s"`, name))
	if amend {
		args = append(args, "--amend")
	}
	gitCommitCmd := exec.Command("git", args...)
	gitCommitCmd.Dir = dir
	var gitCommitCmdStderr bytes.Buffer
	gitCommitCmd.Stderr = &gitCommitCmdStderr
	if err := gitCommitCmd.Run(); err != nil {
		t.Fatal(gitCommitCmdStderr.String())
	}
}

func GitStash(t *testing.T, dir string) {
	gitStashCmd := exec.Command("git", "stash")
	gitStashCmd.Dir = dir
	if err := gitStashCmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func CommitNewFile(t *testing.T, dir string, name string) {
	MkFile(t, dir, name)
	AddAndCommit(t, dir, name)
}

func SetGitRemoteOriginUrl(t *testing.T, dir string, url string) {
	gitRemoteCmd := exec.Command("git", "remote", "set-url", "origin", url)
	gitRemoteCmd.Dir = dir
	var gitCommitCmdStderr bytes.Buffer
	gitRemoteCmd.Stderr = &gitCommitCmdStderr
	if err := gitRemoteCmd.Run(); err != nil {
		t.Fatal(gitCommitCmdStderr.String())
	}
}
