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
	Error   string
	Status  SyncStatus
}

type SyncOptions struct {
	DetailLocalChanges bool
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
	log.Printf("[DEBUG] NewWorkspace with %d arg %s and %d scanned %s\n", len(repositories), util.PluralPrint("repo", len(repositories)), len(scannedRepos), util.PluralPrint("repo", len(scannedRepos)))
	return &Workspace{
		RootDir:      rootDir,
		Repositories: reposAsMap,
	}
}

// Sync performs clones and pulls to sync all Repository instances within a Workspace using the git.Clone and git.Pull APIs.
// A git clone will be performed for any repositories configured within the Workspace that are not present on disk.
// For Workspace repositories already cloned, the Sync operation will perform a git pull.
func (w *Workspace) Sync(syncOptions *SyncOptions) <-chan *SyncUpdate {
	if syncOptions == nil {
		syncOptions = &SyncOptions{}
	}
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
					default:
						c <- &SyncUpdate{Repo: repo.Name, Op: CloneSync, Status: SyncFailure, Error: s.Error}
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
					if s.Status == Pulling {
						continue
					}
					var messages []string
					status := SyncSuccess
					if s.PulledCommits > 0 {
						messages = append(messages, fmt.Sprintf("pulled %d %s", s.PulledCommits, util.PluralPrint("commit", s.PulledCommits)))
					}
					if s.RepoState.LocalCommits > 0 {
						status = SyncWarning
						messages = append(messages, fmt.Sprintf("%d local %s", s.RepoState.LocalCommits, util.PluralPrint("commit", s.RepoState.LocalCommits)))
					}
					localChangesCount := 0
					if syncOptions.DetailLocalChanges {
						var localChangesDetail []string
						if s.RepoState.StagedChanges > 0 {
							localChangesCount += s.RepoState.StagedChanges
							localChangesDetail = append(localChangesDetail, fmt.Sprintf("%d staged", s.RepoState.StagedChanges))
						}
						if s.RepoState.UnstagedChanges > 0 {
							localChangesCount += s.RepoState.UnstagedChanges
							localChangesDetail = append(localChangesDetail, fmt.Sprintf("%d not staged", s.RepoState.UnstagedChanges))
						}
						if s.RepoState.UntrackedFiles > 0 {
							localChangesCount += s.RepoState.UntrackedFiles
							localChangesDetail = append(localChangesDetail, fmt.Sprintf("%d untracked", s.RepoState.UntrackedFiles))
						}
						if localChangesCount > 0 {
							status = SyncWarning
							messages = append(messages, fmt.Sprintf("%d local %s (%s)", localChangesCount, util.PluralPrint("change", localChangesCount), strings.Join(localChangesDetail, ", ")))
						}
						if s.RepoState.BranchesDiverged {
							status = SyncWarning
							messages = append(messages, "local and remote HAVE DIVERGED!")
						}
					} else {
						localChangesCount += s.RepoState.StagedChanges + s.RepoState.UnstagedChanges + s.RepoState.UntrackedFiles
						if localChangesCount > 0 {
							status = SyncWarning
							messages = append(messages, fmt.Sprintf("%d local %s", localChangesCount, util.PluralPrint("change", localChangesCount)))
						}
					}
					stashedChangesCount := len(s.StashList)
					if stashedChangesCount != 0 {
						status = SyncWarning
						messages = append(messages, fmt.Sprintf("%d stashed %s", stashedChangesCount, util.PluralPrint("change", stashedChangesCount)))
					}
					if s.Status != Pulled {
						status = SyncFailure
					}
					message := strings.Join(messages, ", ")
					c <- &SyncUpdate{Repo: repo.Name, Op: PullSync, Status: status, Message: message, Error: s.Error}
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
