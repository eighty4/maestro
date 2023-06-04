package git

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
	Status CloneStatus
	Error  string
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
	Error         string
	PulledCommits int
	RepoState     *RepoState
	StashList     []*StashedChangeset
}

// RepoState holds data from `git status` and is available with git.Status.
type RepoState struct {
	LocalCommits    int
	StagedChanges   int
	UnstagedChanges int
	UntrackedFiles  int
}

type StashedChangeset struct {
	Description  string
	Name         string
	OnBranch     string
	OnCommitHash string
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
				update = &CloneUpdate{Status: CloneFailed, Error: err.Error()}
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
		return &CloneUpdate{Status: CloneFailed, Error: cloneConnectionFailureMsg}
	} else if strings.Contains(stderr, cloneAuthFailedErr) {
		return &CloneUpdate{Status: AuthRequired, Error: cloneAuthFailedMsg}
	} else if strings.Contains(stderr, cloneRepoNotFoundErr) {
		return &CloneUpdate{Status: CloneRepoNotFound, Error: cloneRepoNotFoundMsg}
	} else if strings.Contains(stderr, cloneBadRequestErr) {
		return &CloneUpdate{Status: BadRedirect, Error: cloneBadRequestMsg}
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
		var update *PullUpdate
		if err == nil {
			update = &PullUpdate{Status: Pulled}
		} else {
			update = makePullErrorUpdate(stderr.String())
			if update == nil {
				log.Printf("[ERROR] git.Pull(%s) unhandled error\nstderr:\n==========\n%s\n==========\n", dir, stderr.String())
				update = &PullUpdate{Status: PullFailed, Error: err.Error()}
			}
		}
		if update.Status != NotRepository {
			wg := sync.WaitGroup{}
			wg.Add(3)
			go func() {
				update.PulledCommits, err = getPulledCommitCount(dir, stdoutStr)
				if err != nil {
					log.Printf("[ERROR] git.Pull(%s) resolve pulled commit count error %s\n", dir, err.Error())
				}
				wg.Done()
			}()
			go func() {
				update.RepoState, err = Status(dir)
				if err != nil {
					log.Printf("[ERROR] git.Status(%s) error %s\n", dir, err.Error())
				}
				wg.Done()
			}()
			go func() {
				update.StashList, err = StashList(dir)
				if err != nil {
					log.Printf("[ERROR] git.StashList(%s) error %s\n", dir, err.Error())
				}
				wg.Done()
			}()
			wg.Wait()
		}
		c <- update
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
			return &PullUpdate{Status: ConnectionFailure, Error: pullConnectionFailureMsg}
		} else {
			connectionFailureCauseEndIndex := connectionFailureErrIndex - 1
			if stderr[connectionFailureCauseEndIndex-1] == '.' {
				connectionFailureCauseEndIndex--
			}
			connectionFailureCause := strings.TrimSpace(stderr[0:connectionFailureCauseEndIndex])
			if connectionFailureCause == pullGitHubRepositoryNotFoundErr {
				return &PullUpdate{Status: PullRepoNotFound, Error: pullRepositoryNotFoundMsg}
			} else {
				return &PullUpdate{Status: ConnectionFailure, Error: fmt.Sprintf(`"%s"`, connectionFailureCause)}
			}
		}
	} else if strings.Contains(stderr, pullOverwritesLocalChangesErr) {
		return &PullUpdate{Status: OverwritesLocalChanges, Error: pullOverwritesLocalChangesMsg}
	} else if strings.Contains(stderr, pullMergeConflictErr) {
		return &PullUpdate{Status: MergeConflict, Error: pullMergeConflictMsg}
	} else if strings.Contains(stderr, pullDetachedHeadErr) {
		return &PullUpdate{Status: DetachedHead, Error: pullDetachedHeadMsg}
	} else if strings.Contains(stderr, pullRemoteBranchNotFoundErr) {
		return &PullUpdate{Status: RemoteBranchNotFound, Error: pullRemoteBranchNotFoundMsg}
	} else if strings.Contains(stderr, pullUnstagedChangesErr) {
		return &PullUpdate{Status: UnstagedChanges, Error: pullUnstagedChangesMsg}
	} else if strings.Contains(stderr, pullUnsetUpstreamErr) {
		return &PullUpdate{Status: UnsetUpstream, Error: pullUnsetUpstreamMsg}
	} else if strings.Contains(stderr, pullCouldNotResolveHostErr) {
		return &PullUpdate{Status: CouldNotResolveHost, Error: pullCouldNotResolveHostMsg}
	} else if strings.Index(stderr, pullNotRepositoryErr) == 0 {
		return &PullUpdate{Status: NotRepository, Error: pullNotRepositoryMsg}
	} else if strings.Index(stderr, pullRepositoryNotFoundErrPre) == 0 && strings.Contains(stderr, pullRepositoryNotFoundErrPost) {
		return &PullUpdate{Status: PullRepoNotFound, Error: pullRepositoryNotFoundMsg}
	} else if stderr[0:7] == "fatal: " {
		return &PullUpdate{Status: PullFailed, Error: stderr[0:7]}
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
	cmtRange := fmt.Sprintf("%s..%s", fromCommitHash, toCommitHash)
	gitCmtCountCmd := exec.Command("git", "rev-list", cmtRange, "--count")
	gitCmtCountCmd.Dir = dir
	var stdout bytes.Buffer
	gitCmtCountCmd.Stdout = &stdout
	gitCmtCountCmd.Stderr = nil
	gitCmtCountCmd.Stderr = nil
	err := gitCmtCountCmd.Run()
	if err != nil {
		return -1, fmt.Errorf("error on 'git rev-list %s --count' (%s)", cmtRange, err.Error())
	}
	commitCount, err := parseRevListCommitCount(stdout.String())
	if err != nil {
		return -1, fmt.Errorf("atoi error on 'git rev-list %s --count' output (%s)", cmtRange, err.Error())
	} else {
		return commitCount, nil
	}
}

func StashList(dir string) ([]*StashedChangeset, error) {
	gitStashListCmd := exec.Command("git", "stash", "list")
	gitStashListCmd.Dir = dir
	var stdoutBuf bytes.Buffer
	gitStashListCmd.Stdout = &stdoutBuf
	gitStashListCmd.Stderr = nil
	err := gitStashListCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error on 'git stash list' (%s)", err.Error())
	}
	stdout := strings.TrimSpace(stdoutBuf.String())
	if len(stdout) == 0 {
		return nil, nil
	}
	lines := strings.Split(stdout, "\n")
	var stashes []*StashedChangeset
	for _, line := range lines {
		lineSplit := strings.Split(line, ":")
		name := lineSplit[0]
		trimmedOnBranchSegment := strings.TrimSpace(lineSplit[1])
		onBranch := trimmedOnBranchSegment[strings.LastIndex(trimmedOnBranchSegment, " ")+1:]
		trimmedDescription := strings.TrimSpace(lineSplit[2])
		spaceAfterCommitHash := strings.Index(trimmedDescription, " ")
		onCommitHash := trimmedDescription[0:spaceAfterCommitHash]
		description := trimmedDescription[spaceAfterCommitHash+1:]
		stashes = append(stashes, &StashedChangeset{
			Description:  description,
			Name:         name,
			OnBranch:     onBranch,
			OnCommitHash: onCommitHash,
		})
	}
	return stashes, nil
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
	localCommits := 0
	stagedChanges := 0
	unstagedChanges := 0
	untrackedFiles := 0
	lines := strings.Split(stdout.String(), "\n")
	matches := regex.FindStringSubmatch(lines[1])
	if len(matches) > 1 {
		localCommits, err = strconv.Atoi(matches[1])
		if err != nil {
			return nil, err
		}
	}
	var tracking string
	for _, line := range lines {
		if strings.Index(line, "Changes to be committed") == 0 {
			tracking = "staged"
		} else if strings.Index(line, "Changes not staged for commit") == 0 {
			tracking = "unstaged"
		} else if strings.Index(line, "Untracked files") == 0 {
			tracking = "untracked"
		} else if len(tracking) == 0 {
			continue
		} else if strings.Index(line, "  (use") == 0 {
			continue
		} else if len(strings.TrimSpace(line)) == 0 {
			continue
		} else if line[0] != '\t' {
			continue
		} else {
			switch tracking {
			case "staged":
				stagedChanges++
				break
			case "unstaged":
				unstagedChanges++
				break
			case "untracked":
				untrackedFiles++
				break
			}
		}
	}
	repoState := &RepoState{
		LocalCommits:    localCommits,
		StagedChanges:   stagedChanges,
		UnstagedChanges: unstagedChanges,
		UntrackedFiles:  untrackedFiles,
	}
	return repoState, nil
}

func getPulledCommitCount(dir string, gitPullStdout string) (int, error) {
	if strings.Index(gitPullStdout, "Updating") != 0 {
		return 0, nil
	}
	from, to, err := parsePulledCommitRange(gitPullStdout)
	if err != nil {
		return -1, err
	}
	return RevListCommitCount(dir, from, to)
}

func parsePulledCommitRange(gitPullStdout string) (string, string, error) {
	regex, err := regexp.Compile(`Updating ([a-z\d]+)\.\.([a-z\d]+)`)
	if err != nil {
		return "", "", fmt.Errorf("err creating regex for git pull commit hashes (%s)", err.Error())
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
	if runtime.GOOS == "windows" {
		gitRevParseStdout = strings.ReplaceAll(gitRevParseStdout, "/", "\\")
	}
	return strings.TrimSpace(gitRevParseStdout)
}
