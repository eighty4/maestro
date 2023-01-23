package main

import (
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"log"
	"os"
	"os/exec"
	"regexp"
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
	Dir        string
	Path       string
	PullState  GitPullState
	PullCount  int
	LocalCount int
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
	outputStr := stdout.String()
	if success {
		wd.PullState = Pulled
		wd.SetLocalCommitCount()
		wd.SetPulledCommitCount(outputStr)
	} else if strings.Contains(outputStr, "CONFLICT") {
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

func (wd *WorkspaceDir) SetPulledCommitCount(gitPullStdout string) {
	if strings.Index(gitPullStdout, "Updating") != 0 {
		return
	}
	regex, err := regexp.Compile("Updating ([a-z\\d]+)\\.\\.([a-z\\d]+)")
	if err != nil {
		log.Fatalln("err creating regex on git pull stdout", err)
	}
	firstLine := gitPullStdout[:strings.Index(gitPullStdout, "\n")]
	matches := regex.FindStringSubmatch(firstLine)
	if len(matches) != 3 {
		return
	}
	from := matches[1]
	to := matches[2]
	cmtRange := fmt.Sprintf("%s..%s", from, to)
	gitCmtCountCmd := exec.Command("git", "rev-list", cmtRange, "--count")
	gitCmtCountCmd.Dir = wd.Dir
	var stdout bytes.Buffer
	gitCmtCountCmd.Stdout = &stdout
	gitCmtCountCmd.Stderr = nil
	_ = gitCmtCountCmd.Run()
	stdoutString := stdout.String()
	cmtCount, err := strconv.Atoi(strings.TrimSpace(stdoutString))
	if err != nil {
		log.Fatalln(wd.Dir, "atoi error", stdoutString, "\ngit rev-list", cmtRange, "--count\n", stdoutString)
	}
	wd.PullCount = cmtCount
}

func (wd *WorkspaceDir) SetLocalCommitCount() {
	gitStatusCmd := exec.Command("git", "status")
	gitStatusCmd.Dir = wd.Dir
	var stdout bytes.Buffer
	gitStatusCmd.Stdout = &stdout
	gitStatusCmd.Stderr = nil
	_ = gitStatusCmd.Run()
	regex, err := regexp.Compile("Your branch is ahead of '.+/.+' by (\\d+) commits?\\.")
	if err != nil {
		log.Fatalln("err creating regex on git status stdout", err)
	}
	secondLine := strings.Split(stdout.String(), "\n")[1]
	matches := regex.FindStringSubmatch(secondLine)
	if len(matches) > 1 {
		cmtCount, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Fatalln("err creating regex on git status stdout", err)
		}
		wd.LocalCount = cmtCount
	}
}

type WorkspaceGitPull struct {
	ctx            *MaestroContext
	repos          []*WorkspaceDir
	reprint        bool
	maxRepoNameLen int
	mutex          sync.Mutex
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
			gp.printPullState()
			wg.Done()
		}(repo)
	}

	wg.Wait()

	for _, repo := range gp.repos {
		if repo.PullState != Pulled {
			color.HiRed("some repositories could not be pulled")
			os.Exit(1)
		}
	}
	fmt.Println("Finished!")
}

func unquoteCodePoint(s string) string {
	result, err := strconv.Unquote(fmt.Sprintf("'%s'", s))
	if err != nil {
		log.Fatalln(err)
	}
	return result
}

func (gp *WorkspaceGitPull) initRepositories() {
	var wg sync.WaitGroup
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
				gp.mutex.Lock()
				gp.repos = append(gp.repos, repo)
				gp.mutex.Unlock()
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
	gp.mutex.Lock()
	defer gp.mutex.Unlock()
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
		repoLen := len(gp.repos)
		if repoLen == 1 {
			fmt.Println("Updating 1 repository")
		} else {
			fmt.Printf("Updating %d repositories\n", len(gp.repos))
		}
	}
	redX := color.New(color.FgRed, color.Bold).Sprint(unquoteCodePoint("\\U00002715"))
	check := unquoteCodePoint("\\U00002714")
	greenCheck := color.New(color.FgGreen, color.Bold).Sprint(check)
	yellowCheck := color.New(color.FgYellow, color.Bold).Sprint(unquoteCodePoint("\\U00002714"))
	fmtStr := fmt.Sprintf("  %%%ds %%s %%s\n", gp.maxRepoNameLen)
	for _, repo := range gp.repos {
		checkOrX := ""
		textMsg := ""
		if repo.PullState == Pulled {
			checkOrX = greenCheck
			if repo.PullCount == 1 {
				textMsg = "pulled 1 commit"
			} else if repo.PullCount > 1 {
				textMsg = fmt.Sprintf("pulled %d commits", repo.PullCount)
			}
			if repo.LocalCount > 0 {
				checkOrX = yellowCheck
				localCmtTextMsg := ""
				if repo.LocalCount == 1 {
					localCmtTextMsg = "1 local commit"
				} else {
					localCmtTextMsg = fmt.Sprintf("%d local commits", repo.LocalCount)
				}
				if len(textMsg) == 0 {
					textMsg = localCmtTextMsg
				} else {
					textMsg = fmt.Sprintf("%s, %s", textMsg, localCmtTextMsg)
				}
			}
		} else if repo.PullState == MergeConflict {
			checkOrX = redX
			textMsg = "merge conflict"
		} else if repo.PullState == UnstagedChanges {
			checkOrX = redX
			textMsg = "unstaged changes"
		}
		fmt.Printf(fmtStr, repo.Dir, checkOrX, textMsg)
	}
}

func directories(ctx *MaestroContext) []*WorkspaceDir {
	var dirs []*WorkspaceDir
	files, err := os.ReadDir(ctx.WorkDir)
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
