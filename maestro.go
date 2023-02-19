package main

import (
	"fmt"
	"github.com/eighty4/maestro/git"
	"github.com/eighty4/maestro/util"
	"github.com/fatih/color"
	"log"
	"os"
	"strconv"
)

func main() {
	if util.IsDebug() {
		color.HiYellow("debug enabled")
	}
	util.InitLogging()

	cfg, err := parseConfig(util.Cwd())
	if err != nil {
		println("config error:\n  " + err.Error())
		os.Exit(1)
	}

	if util.IsDebug() && cfg != nil {
		log.Printf("%s has %d repositories", cfg.Filename, len(cfg.Repositories))
	}

	if len(os.Args) > 1 && os.Args[1] == "git" {
		gitSync(cfg)
	} else {
		println("run 'maestro git' to perform a sync of your local workspace")
		os.Exit(1)
	}
}

func gitSync(cfg *Config) {
	var repositories []*git.Repository
	if cfg != nil {
		repositories = cfg.Repositories
	}
	ws := git.NewWorkspace(util.Cwd(), repositories, 2)

	// calc max len of a repo name for print formatting
	maxNameLen := 0
	for _, repo := range ws.Repositories {
		nameLen := len(repo.Name)
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
	}

	up := NewUnicodePrinting()
	// escapes %% to % so given maxNameLen == 3, fmtStr := "  %3s %s\n"
	fmtStr := fmt.Sprintf("  %%%ds %%s %%s\n", maxNameLen)

	printStatusUpdate := func(s *git.SyncUpdate) {
		var checkOrX string
		switch s.Status {
		case git.SyncSuccess:
			checkOrX = up.greenCheck
			break
		case git.SyncWarning:
			checkOrX = up.yellowCheck
			break
		case git.SyncFailure:
			checkOrX = up.redX
			break
		}
		fmt.Printf(fmtStr, s.Repo, checkOrX, s.Message)
	}

	println(fmt.Sprintf("Syncing %d repositories", len(ws.Repositories)))
	c := ws.Sync()
	for {
		update, ok := <-c
		if ok {
			printStatusUpdate(update)
		} else {
			println("Done!")
			break
		}
	}
}

type UnicodePrinting struct {
	greenCheck  string
	yellowCheck string
	redX        string
}

func NewUnicodePrinting() UnicodePrinting {
	unquoteCodePoint := func(s string) string {
		result, err := strconv.Unquote(fmt.Sprintf("'%s'", s))
		if err != nil {
			log.Fatalln(err)
		}
		return result
	}
	check := unquoteCodePoint("\\U00002714")
	greenCheck := color.New(color.FgGreen, color.Bold).Sprint(check)
	yellowCheck := color.New(color.FgYellow, color.Bold).Sprint(check)
	redX := color.New(color.FgRed, color.Bold).Sprint(unquoteCodePoint("\\U00002715"))
	return UnicodePrinting{
		greenCheck:  greenCheck,
		yellowCheck: yellowCheck,
		redX:        redX,
	}
}
