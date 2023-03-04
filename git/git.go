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
	Cloning           CloneStatus = "Cloning"
	Cloned            CloneStatus = "Cloned"
	BadRedirect       CloneStatus = "BadRedirect"
	CloneRepoNotFound CloneStatus = "RepositoryNotFound"
	AuthRequired      CloneStatus = "AuthRequired"
	CloneFailed       CloneStatus = "CloneFailed"
)

type CloneUpdate struct {
	Status  CloneStatus
	Message string
}

type PullStatus string

const (
	Pulling                PullStatus = "Pulling"
	Pulled                 PullStatus = "Pulled"
	CouldNotResolveHost    PullStatus = "CouldNotResolveHost"
	ConnectionFailure      PullStatus = "ConnectionFailure"
	OverwritesLocalChanges PullStatus = "OverwritesLocalChanges"
	MergeConflict          PullStatus = "MergeConflict"
	DetachedHead           PullStatus = "DetachedHead"
	PullRepoNotFound       PullStatus = "RepositoryNotFound"
	RemoteBranchNotFound   PullStatus = "RemoteBranchNotFound"
	UnsetUpstream          PullStatus = "UnsetUpstream"
	UnstagedChanges        PullStatus = "UnstagedChanges"
	NotRepository          PullStatus = "NotRepository"
	PullFailed             PullStatus = "PullFailed"
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
		// GIT_TERMINAL_PROMPT=0 prevents stdin prompting for http auth credentials
		gitCloneCmd.Env = []string{"GIT_TERMINAL_PROMPT=0"}
		var stderr bytes.Buffer
		gitCloneCmd.Stdout = nil
		gitCloneCmd.Stderr = &stderr
		err := gitCloneCmd.Run()
		if err == nil {
			c <- &CloneUpdate{Status: Cloned}
		} else {
			update := makeCloneErrorUpdate(stderr.String())
			if update == nil {
				log.Printf("[ERROR] git.Clone(%s, %s) unhandled error; stderr: %s", dir, url, stderr.String())
				update = &CloneUpdate{Status: CloneFailed, Message: err.Error()}
			}
			c <- update
		}
		close(c)
	}()
	return c
}

const (
	cloneConnectionFailureErr = "fatal: Could not read from remote repository."
	cloneConnectionFailureMsg = "connection failure with remote repository"
	cloneRepoNotFoundErr      = "not found"
	cloneRepoNotFoundMsg      = "repository not found"
	cloneAuthFailedErr        = "fatal: could not read Username for"
	cloneAuthFailedMsg        = "authentication required"
	cloneBadRequestErr        = "fatal: unable to update url base from redirection:"
	cloneBadRequestMsg        = "following http redirect did not connect to a git repository"
)

func makeCloneErrorUpdate(stderr string) *CloneUpdate {
	if strings.Contains(stderr, cloneConnectionFailureErr) {
		return &CloneUpdate{Status: CloneFailed, Message: cloneConnectionFailureMsg}
	} else if strings.Contains(stderr, cloneAuthFailedErr) {
		return &CloneUpdate{Status: AuthRequired, Message: cloneAuthFailedMsg}
	} else if strings.Contains(stderr, cloneRepoNotFoundErr) {
		return &CloneUpdate{Status: CloneRepoNotFound, Message: cloneRepoNotFoundMsg}
	} else if strings.Contains(stderr, cloneBadRequestErr) {
		return &CloneUpdate{Status: BadRedirect, Message: cloneBadRequestMsg}
	} else {
		return nil
	}
}

func Pull(dir string) <-chan *PullUpdate {
	c := make(chan *PullUpdate)
	go func() {
		c <- &PullUpdate{Status: Pulling}
		gitPullCmd := exec.Command("git", "pull", "--ff-only")
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
			update := makePullErrorUpdate(stderr.String())
			if update == nil {
				log.Printf("[ERROR] git.Pull(%s) unhandled error\nstderr:\n==========\n%s\n==========\n", dir, stderr.String())
				update = &PullUpdate{Status: PullFailed, Message: err.Error()}
			}
			c <- update
		}
		close(c)
	}()
	return c
}

const (
	pullConnectionFailureErr = "fatal: Could not read from remote repository."
	pullConnectionFailureMsg = "connection failure with remote repository"

	pullGitHubRepositoryNotFoundErr = "ERROR: Repository not found"

	pullDetachedHeadErr = "You are not currently on a branch."
	pullDetachedHeadMsg = "detached from a branch"

	pullOverwritesLocalChangesErr = "Your local changes to the following files would be overwritten by merge:"
	pullOverwritesLocalChangesMsg = "local changes would be overwritten"

	pullMergeConflictErr = "fatal: Not possible to fast-forward, aborting."
	pullMergeConflictMsg = "unable to pull without a merge or interactive rebase"

	pullRemoteBranchNotFoundErr = "from the remote, but no such ref was fetched."
	pullRemoteBranchNotFoundMsg = "tracking branch not found on remote"

	pullUnstagedChangesErr = "You have unstaged changes."
	pullUnstagedChangesMsg = "unstaged changes"

	pullUnsetUpstreamErr = "There is no tracking information for the current branch."
	pullUnsetUpstreamMsg = "not tracking an upstream remote"

	pullRepositoryNotFoundErrPre  = "fatal: repository"
	pullRepositoryNotFoundErrPost = "not found"
	pullRepositoryNotFoundMsg     = "repository not found"

	pullNotRepositoryErr = "fatal: not a git repository (or any of the parent directories): .git"
	pullNotRepositoryMsg = "not a repository"

	pullCouldNotResolveHostErr = "Could not resolve host: "
	pullCouldNotResolveHostMsg = "could not resolve host"
)

func makePullErrorUpdate(stderr string) *PullUpdate {
	if strings.Contains(stderr, pullConnectionFailureErr) {
		connectionFailureErrIndex := strings.Index(stderr, pullConnectionFailureErr)
		if connectionFailureErrIndex == 0 {
			return &PullUpdate{Status: ConnectionFailure, Message: pullConnectionFailureMsg}
		} else {
			connectionFailureCauseEndIndex := connectionFailureErrIndex - 1
			if stderr[connectionFailureCauseEndIndex-1] == '.' {
				connectionFailureCauseEndIndex--
			}
			connectionFailureCause := strings.TrimSpace(stderr[0:connectionFailureCauseEndIndex])
			if connectionFailureCause == pullGitHubRepositoryNotFoundErr {
				return &PullUpdate{Status: PullRepoNotFound, Message: pullRepositoryNotFoundMsg}
			} else {
				return &PullUpdate{Status: ConnectionFailure, Message: fmt.Sprintf(`"%s"`, connectionFailureCause)}
			}
		}
	} else if strings.Contains(stderr, pullOverwritesLocalChangesErr) {
		return &PullUpdate{Status: OverwritesLocalChanges, Message: pullOverwritesLocalChangesMsg}
	} else if strings.Contains(stderr, pullMergeConflictErr) {
		return &PullUpdate{Status: MergeConflict, Message: pullMergeConflictMsg}
	} else if strings.Contains(stderr, pullDetachedHeadErr) {
		return &PullUpdate{Status: DetachedHead, Message: pullDetachedHeadMsg}
	} else if strings.Contains(stderr, pullRemoteBranchNotFoundErr) {
		return &PullUpdate{Status: RemoteBranchNotFound, Message: pullRemoteBranchNotFoundMsg}
	} else if strings.Contains(stderr, pullUnstagedChangesErr) {
		return &PullUpdate{Status: UnstagedChanges, Message: pullUnstagedChangesMsg}
	} else if strings.Contains(stderr, pullUnsetUpstreamErr) {
		return &PullUpdate{Status: UnsetUpstream, Message: pullUnsetUpstreamMsg}
	} else if strings.Contains(stderr, pullCouldNotResolveHostErr) {
		return &PullUpdate{Status: CouldNotResolveHost, Message: pullCouldNotResolveHostMsg}
	} else if strings.Index(stderr, pullNotRepositoryErr) == 0 {
		return &PullUpdate{Status: NotRepository, Message: pullNotRepositoryMsg}
	} else if strings.Index(stderr, pullRepositoryNotFoundErrPre) == 0 && strings.Index(stderr, pullRepositoryNotFoundErrPost) != -1 {
		return &PullUpdate{Status: PullRepoNotFound, Message: pullRepositoryNotFoundMsg}
	} else {
		return nil
	}
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
