package main

import (
	"fmt"
	"runtime"
)

type GradleTaskConfig struct {
	Module string
	Task   string
}

func (c *GradleTaskConfig) CreateProcess(context *MaestroContext) *Process {
	var process *Process
	args := []string{"-q", "--console=plain", fmt.Sprintf("%s:%s", c.Module, c.Task)}
	if runtime.GOOS == "windows" {
		process = NewProcess(".\\gradlew", args, context.WorkDir)
	} else {
		process = NewProcess("./gradlew", args, context.WorkDir)
	}
	process.Logging.print = true
	return process
}
