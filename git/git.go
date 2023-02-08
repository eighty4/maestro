package git

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type CloneStatus string

const (
	Cloning     CloneStatus = "Cloning"
	Cloned      CloneStatus = "Cloned"
	CloneFailed CloneStatus = "CloneFailed"
)

type CloneUpdate struct {
	Status  CloneStatus
	Message string
}

type PullStatus string

const (
	Pulling              PullStatus = "Pulling"
	Pulled               PullStatus = "Pulled"
	CouldNotResolveHost  PullStatus = "CouldNotResolveHost"
	ConnectionFailure    PullStatus = "ConnectionFailure"
	MergeConflict        PullStatus = "MergeConflict"
	DetachedHead         PullStatus = "DetachedHead"
	DivergentBranches    PullStatus = "DivergentBranches"
	RepositoryNotFound   PullStatus = "RepositoryNotFound"
	RemoteBranchNotFound PullStatus = "RemoteBranchNotFound"
	UnsetUpstream        PullStatus = "UnsetUpstream"
	UnstagedChanges      PullStatus = "UnstagedChanges"
	PullFailed           PullStatus = "PullFailed"
)

type PullUpdate struct {
	Status        PullStatus
	Message       string
	PulledCommits int
	RepoState     *RepoState
}

// RepoState holds data from `git status` and is available with git.Status.
type RepoState struct {
	LocalCommits int
}

func Clone(dir string, url string) <-chan *CloneUpdate {
	c := make(chan *CloneUpdate)
	go func() {
		c <- &CloneUpdate{Status: Cloning}
		gitCloneCmd := exec.Command("git", "clone", url, dir)
		var stderr bytes.Buffer
		gitCloneCmd.Stdout = nil
		gitCloneCmd.Stderr = &stderr
		err := gitCloneCmd.Run()
		if err == nil {
			c <- &CloneUpdate{Status: Cloned}
		} else {
			stderrStr := stderr.String()
			if strings.Contains(stderrStr, cloneRepoNotFoundErr) {
				c <- &CloneUpdate{Status: CloneFailed, Message: cloneRepoNotFoundMsg}
			} else {
				log.Printf("[ERROR] git.Clone(%s, %s) unhandled error; stderr: %s", dir, url, stderrStr)
				c <- &CloneUpdate{Status: CloneFailed, Message: err.Error()}
			}
		}
		close(c)
	}()
	return c
}

const (
	cloneRepoNotFoundErr = "ERROR: Repository not found"
	cloneRepoNotFoundMsg = "repository not found"
)

func Pull(dir string) <-chan *PullUpdate {
	// todo parameterize with Default, Merge, Rebase and FF-only behaviors
	c := make(chan *PullUpdate)
	go func() {
		c <- &PullUpdate{Status: Pulling}
		gitPullCmd := exec.Command("git", "pull")
		gitPullCmd.Dir = dir
		var stdout bytes.Buffer
		gitPullCmd.Stdout = &stdout
		var stderr bytes.Buffer
		gitPullCmd.Stderr = &stderr
		err := gitPullCmd.Run()
		stdoutStr := stdout.String()
		if err == nil {
			pulledCommitCount, err := getPulledCommitCount(dir, stdoutStr)
			if err != nil {
				log.Printf("[ERROR] git.Pull(%s) resolve pulled commit count error %s\n", dir, err.Error())
			}
			s, err := Status(dir)
			if err != nil {
				log.Printf("[ERROR] git.Status(%s) error %s\n", dir, err.Error())
			}
			c <- &PullUpdate{Status: Pulled, PulledCommits: pulledCommitCount, RepoState: s}
		} else {
			stderrStr := stderr.String()
			println(stderrStr)
			if strings.Contains(stderrStr, pullConnectionFailureErr) {
				errI := strings.Index(stderrStr, pullConnectionFailureErr)
				if errI == 0 {
					c <- &PullUpdate{Status: ConnectionFailure, Message: pullConnectionFailureMsg}
				} else {
					substrEnd := errI - 1
					if stderrStr[substrEnd-1] == '.' {
						substrEnd--
					}
					message := stderrStr[0:substrEnd]
					if message == pullGitHubRepositoryNotFoundErr {
						c <- &PullUpdate{Status: RepositoryNotFound, Message: pullRepositoryNotFoundMsg}
					} else {
						c <- &PullUpdate{Status: ConnectionFailure, Message: fmt.Sprintf(`"%s"`, message)}
					}
				}
			} else if strings.Contains(stderrStr, pullMergeConflictErr) {
				c <- &PullUpdate{Status: MergeConflict, Message: pullMergeConflictMsg}
			} else if strings.Contains(stdoutStr, pullRebaseConflictErr) {
				_ = RebaseAbort(dir)
				c <- &PullUpdate{Status: MergeConflict, Message: pullRebaseConflictMsg}
			} else if strings.Contains(stderrStr, pullDivergentBranchesErr) {
				c <- &PullUpdate{Status: DivergentBranches, Message: pullDivergentBranchesMsg}
			} else if strings.Contains(stderrStr, pullDetachedHeadErr) {
				c <- &PullUpdate{Status: DetachedHead, Message: pullDetachedHeadMsg}
			} else if strings.Contains(stderrStr, pullRemoteBranchNotFoundErr) {
				c <- &PullUpdate{Status: RemoteBranchNotFound, Message: pullRemoteBranchNotFoundMsg}
			} else if strings.Contains(stderrStr, pullUnstagedChangesErr) {
				c <- &PullUpdate{Status: UnstagedChanges, Message: pullUnstagedChangesMsg}
			} else if strings.Contains(stderrStr, pullUnsetUpstreamErr) {
				c <- &PullUpdate{Status: UnsetUpstream, Message: pullUnsetUpstreamMsg}
			} else if strings.Contains(stderrStr, pullCouldNotResolveHostMsg) {
				c <- &PullUpdate{Status: CouldNotResolveHost, Message: "asdf"}
			} else if strings.Index(stderrStr, pullRepositoryNotFoundErrPre) == 0 && strings.Index(stderrStr, pullRepositoryNotFoundErrPost) != -1 {
				c <- &PullUpdate{Status: RepositoryNotFound, Message: pullRepositoryNotFoundMsg}
			} else {
				log.Printf("[ERROR] git.Pull(%s) unhandled error; stderr: %s", dir, stderrStr)
				c <- &PullUpdate{Status: PullFailed, Message: err.Error()}
			}
		}
		close(c)
	}()
	return c
}

const (
	pullConnectionFailureErr        = "fatal: Could not read from remote repository."
	pullConnectionFailureMsg        = "connection failure with remote repository"
	pullGitHubRepositoryNotFoundErr = "ERROR: Repository not found"
	pullDetachedHeadErr             = "You are not currently on a branch."
	pullDetachedHeadMsg             = "detached from a branch"
	pullDivergentBranchesErr        = "You have divergent branches and need to specify how to reconcile them."
	pullDivergentBranchesMsg        = "divergent branches (require a merge or rebase)"
	pullMergeConflictErr            = "Your local changes to the following files would be overwritten by merge:"
	pullMergeConflictMsg            = "merge conflict"
	pullRebaseConflictErr           = "CONFLICT"
	pullRebaseConflictMsg           = "merge conflict (don't worry, rebase was aborted)"
	pullRemoteBranchNotFoundErr     = "from the remote, but no such ref was fetched."
	pullRemoteBranchNotFoundMsg     = "tracking branch not found on remote"
	pullUnstagedChangesErr          = "You have unstaged changes."
	pullUnstagedChangesMsg          = "unstaged changes"
	pullUnsetUpstreamErr            = "There is no tracking information for the current branch."
	pullUnsetUpstreamMsg            = "not tracking an upstream remote"
	pullRepositoryNotFoundErrPre    = "fatal: repository"
	pullRepositoryNotFoundErrPost   = "not found"
	pullRepositoryNotFoundMsg       = "repository not found"
	pullCouldNotResolveHostErr      = "Could not resolve host: "
	pullCouldNotResolveHostMsg      = "could not resolve host"
)

func RebaseAbort(dir string) error {
	gitPullCmd := exec.Command("git", "rebase", "--abort")
	gitPullCmd.Dir = dir
	return gitPullCmd.Run()
}

// RevParseShowTopLevel returns absolute path of top level directory of Git repository.
// This command will return `/repo` if called from `/repo/subdir` if `/repo` is a Git repository.
func RevParseShowTopLevel(dir string) (string, error) {
	gitRevParseCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	gitRevParseCmd.Dir = dir
	var stdout bytes.Buffer
	gitRevParseCmd.Stdout = &stdout
	gitRevParseCmd.Stderr = nil
	err := gitRevParseCmd.Run()
	if err != nil {
		return "", err
	} else {
		return parseRevParseAbsolutePath(stdout.String()), nil
	}
}

func RevListCommitCount(dir string, fromCommitHash string, toCommitHash string) (int, error) {
	// todo test
	cmtRange := fmt.Sprintf("%s..%s", fromCommitHash, toCommitHash)
	gitCmtCountCmd := exec.Command("git", "rev-list", cmtRange, "--count")
	gitCmtCountCmd.Dir = dir
	var stdout bytes.Buffer
	gitCmtCountCmd.Stdout = &stdout
	gitCmtCountCmd.Stderr = nil
	gitCmtCountCmd.Stderr = nil
	err := gitCmtCountCmd.Run()
	if err != nil {
		return -1, errors.New(fmt.Sprintf("error on 'git rev-list %s --count' (%s)", cmtRange, err.Error()))
	}
	commitCount, err := parseRevListCommitCount(stdout.String())
	if err != nil {
		return -1, errors.New(fmt.Sprintf("atoi error on 'git rev-list %s --count' output (%s)", cmtRange, err.Error()))
	} else {
		return commitCount, nil
	}
}

func Status(dir string) (*RepoState, error) {
	gitStatusCmd := exec.Command("git", "status")
	gitStatusCmd.Dir = dir
	var stdout bytes.Buffer
	gitStatusCmd.Stdout = &stdout
	gitStatusCmd.Stderr = nil
	_ = gitStatusCmd.Run()
	regex, err := regexp.Compile(`Your branch is ahead of '.+/.+' by (\d+) commits?\.`)
	if err != nil {
		return nil, err
	}
	secondLine := strings.Split(stdout.String(), "\n")[1]
	matches := regex.FindStringSubmatch(secondLine)
	if len(matches) > 1 {
		localCommits, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, err
		} else {
			return &RepoState{LocalCommits: localCommits}, nil
		}
	} else {
		return &RepoState{LocalCommits: 0}, nil
	}
}

func getPulledCommitCount(dir string, gitPullStdout string) (int, error) {
	// todo test
	if strings.Index(gitPullStdout, "Updating") != 0 {
		return 0, nil
	}
	from, to, err := parsePullCommitRange(gitPullStdout)
	if err != nil {
		return -1, err
	}
	return RevListCommitCount(dir, from, to)
}

func parsePullCommitRange(gitPullStdout string) (string, string, error) {
	regex, err := regexp.Compile(`Updating ([a-z\d]+)\.\.([a-z\d]+)`)
	if err != nil {
		return "", "", errors.New(fmt.Sprintf("err creating regex for git pull commit hashes (%s)", err.Error()))
	}
	firstLine := gitPullStdout[:strings.Index(gitPullStdout, "\n")]
	matches := regex.FindStringSubmatch(firstLine)
	if len(matches) != 3 {
		return "", "", errors.New("failed to find commit hashes in git pull stdout")
	}
	from := matches[1]
	to := matches[2]
	return from, to, nil
}

func parseRevListCommitCount(gitRevListStdout string) (int, error) {
	commitCount, err := strconv.Atoi(strings.TrimSpace(gitRevListStdout))
	if err != nil {
		return -1, err
	}
	return commitCount, nil
}

func parseRevParseAbsolutePath(gitRevParseStdout string) string {
	return strings.TrimSpace(gitRevParseStdout)
}
