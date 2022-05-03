package main

import (
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type GitPullState uint8

const (
	Pulling = iota
	Pulled
	MergeConflict
	UnstagedChanges
)

type WorkspaceDir struct {
	Dir       string
	Path      string
	PullState GitPullState
}

func NewWorkspaceDir(ctx *MaestroContext, dirName string) WorkspaceDir {
	strings.Join([]string{ctx.WorkDir, dirName}, string(os.PathSeparator))
	return WorkspaceDir{
		Dir:  dirName,
		Path: strings.Join([]string{ctx.WorkDir, dirName}, string(os.PathSeparator)),
	}
}

func (wd *WorkspaceDir) IsGitRepo() bool {
	gitStatusCmd := exec.Command("git", "status")
	gitStatusCmd.Dir = wd.Dir
	gitStatusCmd.Stdout = nil
	gitStatusCmd.Stderr = nil
	err := gitStatusCmd.Run()
	isGitRepo := err == nil && gitStatusCmd.ProcessState.ExitCode() == 0
	if !isGitRepo {
		return false
	} else {
		return wd.IsTopLevelGitRepoDir()
	}
}

func (wd *WorkspaceDir) IsTopLevelGitRepoDir() bool {
	gitRevParseCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	gitRevParseCmd.Dir = wd.Dir
	var stdout bytes.Buffer
	gitRevParseCmd.Stdout = &stdout
	gitRevParseCmd.Stderr = nil
	err := gitRevParseCmd.Run()
	if err != nil {
		return false
	} else {
		repoTopLevelPath := strings.TrimSpace(stdout.String())
		return wd.Path == repoTopLevelPath
	}
}

func (wd *WorkspaceDir) GitPull() {
	gitPullCmd := exec.Command("git", "pull")
	gitPullCmd.Dir = wd.Dir
	var stdout bytes.Buffer
	gitPullCmd.Stdout = &stdout
	gitPullCmd.Stderr = nil
	err := gitPullCmd.Run()
	success := err == nil && gitPullCmd.ProcessState.ExitCode() == 0
	if success {
		wd.PullState = Pulled
	} else if strings.Contains(stdout.String(), "CONFLICT") {
		gitPullCmd := exec.Command("git", "rebase", "--abort")
		gitPullCmd.Dir = wd.Dir
		gitPullCmd.Stdout = nil
		gitPullCmd.Stderr = nil
		_ = gitPullCmd.Run()
		wd.PullState = MergeConflict
	} else {
		wd.PullState = UnstagedChanges
	}
}

type WorkspaceGitPull struct {
	ctx            *MaestroContext
	repos          []*WorkspaceDir
	reprint        bool
	maxRepoNameLen int
}

func NewWorkspaceGitPull(ctx *MaestroContext) *WorkspaceGitPull {
	return &WorkspaceGitPull{
		ctx:     ctx,
		reprint: false,
	}
}

func (gp *WorkspaceGitPull) pull() {
	var wg sync.WaitGroup
	gp.initRepositories()
	gp.printPullState()

	wg.Add(len(gp.repos))
	for _, repo := range gp.repos {
		go func(repo *WorkspaceDir) {
			repo.GitPull()
			var mu sync.Mutex
			mu.Lock()
			gp.printPullState()
			mu.Unlock()
			wg.Done()
		}(repo)
	}

	wg.Wait()

	gp.printPullState()
	for _, repo := range gp.repos {
		if repo.PullState != Pulled {
			color.HiRed("some repositories could not be pulled due to un-staged changes")
			os.Exit(1)
		}
	}
	fmt.Println("Finished!")
}

func unquoteCodePoint(s string) string {
	r, err := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)
	if err != nil {
		log.Fatalln(err)
	}
	return string(r)
}

func (gp *WorkspaceGitPull) initRepositories() {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var maxRepoNameLen = 0

	for _, repo := range directories(gp.ctx) {
		wg.Add(1)
		go func(repo *WorkspaceDir) {
			if repo.IsGitRepo() {
				repoNameLen := len(repo.Dir)
				if repoNameLen > maxRepoNameLen {
					maxRepoNameLen = repoNameLen
				}
				repo.PullState = Pulling
				mu.Lock()
				gp.repos = append(gp.repos, repo)
				mu.Unlock()
			}
			wg.Done()
		}(repo)
	}
	wg.Wait()

	gp.maxRepoNameLen = maxRepoNameLen
	sort.Slice(gp.repos, func(i int, j int) bool {
		return gp.repos[i].Dir < gp.repos[j].Dir
	})
}

func (gp *WorkspaceGitPull) printPullState() {
	if gp.reprint {
		repoLen := len(gp.repos)
		upLine := "\033[A"
		clearLine := "\033[2K"
		for i := 1; i <= repoLen; i++ {
			fmt.Print(upLine)
			fmt.Print(clearLine)
		}
	} else {
		gp.reprint = true
		fmt.Printf("Updating %d repositories\n", len(gp.repos))
	}
	x := color.New(color.FgRed, color.Bold).SprintFunc()(unquoteCodePoint("\\U00002715"))
	check := color.New(color.FgGreen, color.Bold).SprintFunc()(unquoteCodePoint("\\U00002714"))
	fmtStr := fmt.Sprintf("  %%%ds %%s %%s\n", gp.maxRepoNameLen)
	for _, repo := range gp.repos {
		checkOrX := ""
		abortMsg := ""
		if repo.PullState == Pulled {
			checkOrX = check
		} else if repo.PullState == MergeConflict {
			checkOrX = x
			abortMsg = "merge conflict"
		} else if repo.PullState == UnstagedChanges {
			checkOrX = x
			abortMsg = "unstaged changes"
		}
		fmt.Printf(fmtStr, repo.Dir, checkOrX, abortMsg)
	}
}

func directories(ctx *MaestroContext) []*WorkspaceDir {
	var dirs []*WorkspaceDir
	files, err := ioutil.ReadDir(ctx.WorkDir)
	if err != nil {
		log.Fatalln("err workspace dir walk", err)
	} else {
		for _, file := range files {
			if file.IsDir() {
				dir := NewWorkspaceDir(ctx, file.Name())
				dirs = append(dirs, &dir)
			}
		}
	}
	return dirs
}
