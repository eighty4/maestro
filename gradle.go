package main

import (
	"fmt"
	"github.com/eighty4/maestro/composable"
	"runtime"
)

type GradleTaskConfig struct {
	Module string
	Task   string
}

func (c *GradleTaskConfig) CreateProcess(context *MaestroContext) *composable.Process {
	var process *composable.Process
	args := []string{"-q", "--console=plain", fmt.Sprintf("%s:%s", c.Module, c.Task)}
	if runtime.GOOS == "windows" {
		process = composable.NewProcess(".\\gradlew", args, context.WorkDir)
	} else {
		process = composable.NewProcess("./gradlew", args, context.WorkDir)
	}
	return process
}
