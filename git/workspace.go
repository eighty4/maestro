package git

import (
	"fmt"
	"github.com/eighty4/maestro/util"
	"log"
	"strings"
	"sync"
)

type SyncOp string

const (
	CloneSync SyncOp = "Clone"
	PullSync  SyncOp = "Pull"
)

type SyncStatus string

const (
	SyncSuccess SyncStatus = "SyncSuccess"
	SyncWarning SyncStatus = "SyncWarning"
	SyncFailure SyncStatus = "SyncFailure"
)

type SyncUpdate struct {
	Repo    string
	Op      SyncOp
	Message string
	Status  SyncStatus
}

// Workspace represents a local directory structure of git repositories.
type Workspace struct {
	RootDir      string
	Repositories map[string]*Repository
}

// NewWorkspace creates a Workspace at a specified rootDir.
// Repositories will be configured with the repositories argument and using ScanForRepositories.
// Use repoScanDepth to specify how deep subdirectories should be scanned for additional repositories.
// repoScanDepth=0 skips subdirectory scanning.
func NewWorkspace(rootDir string, repositories []*Repository, repoScanDepth int) *Workspace {
	// todo add error to signature and return validation errors
	reposAsMap := make(map[string]*Repository)
	for _, repo := range repositories {
		reposAsMap[repo.Dir] = repo
	}
	var scannedRepos []*Repository
	if repoScanDepth > 0 {
		scannedRepos = ScanForRepositories(rootDir, repoScanDepth)
		for _, repo := range scannedRepos {
			reposAsMap[repo.Dir] = repo
		}
	}
	log.Printf("[DEBUG] NewWorkspace with %d arg repos and %d scanned repos\n", len(repositories), len(scannedRepos))
	return &Workspace{
		RootDir:      rootDir,
		Repositories: reposAsMap,
	}
}

// Sync performs clones and pulls to sync all Repository instances within a Workspace using the git.Clone and git.Pull APIs.
// A git clone will be performed for any repositories configured within the Workspace that are not present on disk.
// For Workspace repositories already cloned, the Sync operation will perform a git pull.
func (w *Workspace) Sync() <-chan *SyncUpdate {
	// todo test
	var clone []*Repository
	var pull []*Repository
	c := make(chan *SyncUpdate)
	wg := sync.WaitGroup{}
	wg.Add(len(w.Repositories))

	for _, repo := range w.Repositories {
		if util.IsDir(repo.Dir) {
			pull = append(pull, repo)
		} else {
			clone = append(clone, repo)
		}
	}

	// todo check newly cloned repos for maestro config to add to workspace and include in ongoing Sync
	go func() {
		for _, repo := range clone {
			repo := repo
			go func() {
				cc := Clone(repo.Dir, repo.Git.Url)
				for {
					s := <-cc
					switch s.Status {
					case Cloning:
						continue
					case Cloned:
						c <- &SyncUpdate{Repo: repo.Name, Op: CloneSync, Status: SyncSuccess, Message: "cloned from " + repo.Git.Url}
						break
					case CloneFailed:
						c <- &SyncUpdate{Repo: repo.Name, Op: CloneSync, Status: SyncFailure, Message: s.Message}
						break
					}
					wg.Done()
					return
				}
			}()
		}
	}()

	go func() {
		for _, repo := range pull {
			repo := repo
			go func() {
				pc := Pull(repo.Dir)
				for {
					s := <-pc
					switch s.Status {
					case Pulling:
						continue
					case Pulled:
						var messages []string
						status := SyncSuccess
						if s.PulledCommits > 0 {
							messages = append(messages, fmt.Sprintf("pulled %d %s", s.PulledCommits, util.PluralPrint("commit", s.PulledCommits)))
						}
						if s.RepoState.LocalCommits > 0 {
							status = SyncWarning
							messages = append(messages, fmt.Sprintf("%d local %s", s.RepoState.LocalCommits, util.PluralPrint("commit", s.RepoState.LocalCommits)))
						}
						if s.RepoState.StagedChanges > 0 {
							status = SyncWarning
							messages = append(messages, fmt.Sprintf("%d staged %s", s.RepoState.StagedChanges, util.PluralPrint("change", s.RepoState.StagedChanges)))
						}
						if s.RepoState.UnstagedChanges > 0 {
							status = SyncWarning
							messages = append(messages, fmt.Sprintf("%d unstaged %s", s.RepoState.UnstagedChanges, util.PluralPrint("change", s.RepoState.UnstagedChanges)))
						}
						if s.RepoState.UntrackedFiles > 0 {
							status = SyncWarning
							messages = append(messages, fmt.Sprintf("%d untracked %s", s.RepoState.UntrackedFiles, util.PluralPrint("file", s.RepoState.UntrackedFiles)))
						}
						var message string
						if len(messages) > 0 {
							message = strings.Join(messages, ", ")
						}
						c <- &SyncUpdate{Repo: repo.Name, Op: PullSync, Status: status, Message: message}
						break
					default:
						c <- &SyncUpdate{Repo: repo.Name, Op: PullSync, Status: SyncFailure, Message: s.Message}
						break
					}
					wg.Done()
					return
				}
			}()
		}
	}()

	go func() {
		wg.Wait()
		close(c)
	}()

	return c
}
