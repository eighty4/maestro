package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func MkDirAndInitGitRepo(t *testing.T, dir string) {
	MkDir(t, dir)
	InitGitRepo(t, dir)
}

func InitGitRepo(t *testing.T, dir string) {
	gitInit := exec.Command("git", "init")
	gitInit.Dir = dir
	gitInit.Stdout = os.Stdout
	gitInit.Stderr = os.Stderr
	if err := gitInit.Run(); err != nil {
		t.Fatal(err)
	}
}

func CloneGitRepo(t *testing.T, dir string, url string) {
	cloneCmd := exec.Command("git", "clone", url, dir)
	err := cloneCmd.Run()
	if err != nil {
		t.Fatal(err)
	} else if cloneCmd.ProcessState.ExitCode() != 0 {
		t.Fatal("git clone error")
	}
}

func GitCheckout(t *testing.T, dir string, branchOrCommitHash string) {
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

func AddAndCommit(t *testing.T, dir string, name string) {
	gitAddCmd := exec.Command("git", "add", name)
	gitAddCmd.Dir = dir
	if err := gitAddCmd.Run(); err != nil {
		t.Fatal(err)
	}
	gitCommitCmd := exec.Command("git", "commit", "-m", fmt.Sprintf(`"%s"`, name))
	gitCommitCmd.Dir = dir
	var gitCommitCmdStderr bytes.Buffer
	gitCommitCmd.Stderr = &gitCommitCmdStderr
	if err := gitCommitCmd.Run(); err != nil {
		t.Fatal(gitCommitCmdStderr.String())
	}
}

func CommitNewFile(t *testing.T, dir string, name string) {
	MkFile(t, dir, name)
	AddAndCommit(t, dir, name)
}
