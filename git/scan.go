package git

import (
	"fmt"
	"github.com/eighty4/maestro/util"
	"log"
	"os"
	"os/exec"
	"path"
	"sync"
)

// ScanForRepositories will recursively look for git repositories within the specified directory.
// repoScanDepth controls how many directories deep the repository scan will recurse.
func ScanForRepositories(dir string, repoScanDepth int) []*Repository {
	log.Printf("[TRACE] ScanForRepositories(\"%s\", %d)\n", dir, repoScanDepth)
	directories, err := subdirectories(dir)
	if err != nil {
		log.Printf("[ERROR] ScanForRepositories(\"%s\", %d) error %s\n", dir, repoScanDepth, err.Error())
		return nil
	} else {
		var repositories []*Repository
		done := make(chan interface{})
		c := make(chan *Repository)
		wg := sync.WaitGroup{}
		wg.Add(len(directories))

		for _, subdirName := range directories {
			subdirAbsPath := path.Join(dir, subdirName)
			subdirName := subdirName
			go func() {
				if isGitRepoRootDir(subdirAbsPath) {
					c <- NewRepository(subdirName, subdirAbsPath, "")
					wg.Done()
				} else if repoScanDepth > 0 {
					go func() {
						repos := ScanForRepositories(subdirAbsPath, repoScanDepth-1)
						if err != nil {
							done <- fmt.Errorf("ScanForRepositories(%s, %d) error: %s", subdirAbsPath, repoScanDepth-1, err.Error())
						} else {
							for _, repo := range repos {
								repo.Name = path.Join(subdirName, repo.Name)
								c <- repo
							}
						}
						wg.Done()
					}()
				} else {
					wg.Done()
				}
			}()
		}

		go func() {
			wg.Wait()
			done <- nil
		}()

		for {
			select {
			case repo := <-c:
				repositories = append(repositories, repo)
				break
			case <-done:
				close(c)
				close(done)
				log.Printf("[DEBUG] ScanForRepositories(\"%s\", %d) found %d %s\n", dir, repoScanDepth, len(repositories), util.SinglePrintIes("repositories", 1))
				return repositories
			}
		}
	}
}

func subdirectories(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	} else {
		var result []string
		for _, file := range files {
			if file.IsDir() {
				result = append(result, file.Name())
			}
		}
		log.Printf("[TRACE] subdirectories(%s) found %d %s\n", dir, len(result), util.PluralPrint("dir", len(result)))
		return result, nil
	}
}

func isGitRepoRootDir(dir string) bool {
	return isGitRepo(dir) && isTopLevelGitRepoDir(dir)
}

func isGitRepo(dir string) bool {
	gitStatusCmd := exec.Command("git", "status")
	gitStatusCmd.Dir = dir
	gitStatusCmd.Stdout = nil
	gitStatusCmd.Stderr = nil
	err := gitStatusCmd.Run()
	return err == nil && gitStatusCmd.ProcessState.ExitCode() == 0
}

func isTopLevelGitRepoDir(dir string) bool {
	repoTopLevelPath, _ := RevParseShowTopLevel(dir)
	return dir == repoTopLevelPath
}
