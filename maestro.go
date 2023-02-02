package main

import (
	"github.com/eighty4/maestro/util"
	"github.com/fatih/color"
	"os"
)

func main() {
	if util.Debug() {
		color.HiYellow("debug enabled")
	}
	util.InitLogging()

	if len(os.Args) > 1 && os.Args[1] == "git" {
		NewWorkspaceGitPull().pull()
	} else {
		println("run 'maestro git' to perform a sync of your local workspace")
		os.Exit(1)
	}
}
