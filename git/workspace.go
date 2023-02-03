package git

import (
	"fmt"
	"github.com/eighty4/maestro/util"
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
	if repoScanDepth > 0 {
		scannedRepos := ScanForRepositories(rootDir, repoScanDepth)
		for _, repo := range scannedRepos {
			reposAsMap[repo.Dir] = repo
		}
	}
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
					case Cloned:
						c <- &SyncUpdate{Repo: repo.Name, Op: CloneSync, Status: SyncSuccess, Message: "cloned in " + repo.Dir}
						break
					case CloneFailed:
						c <- &SyncUpdate{Repo: repo.Name, Op: CloneSync, Status: SyncFailure, Message: s.Message}
						break
					}
				}
			}()
			wg.Done()
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
						message := ""
						status := SyncSuccess
						if s.PulledCommits > 0 {
							message = fmt.Sprintf("pulled %d commits", s.PulledCommits)
						}
						if s.RepoState.LocalCommits > 0 {
							status = SyncWarning
							localCommitMessage := fmt.Sprintf("%d local commits", s.RepoState.LocalCommits)
							if message == "" {
								message = localCommitMessage
							} else {
								strings.Join([]string{message, localCommitMessage}, ", ")
							}
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
