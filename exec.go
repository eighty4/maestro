package main

import (
	"github.com/eighty4/maestro/composable"
	"strings"
)

type ExecConfig struct {
	Cmd string
}

func (c *ExecConfig) CreateProcess(context *MaestroContext) *composable.Process {
	binary, args := ParseExecString(c.Cmd)
	process := composable.NewProcess(binary, args, context.WorkDir)
	return process
}

func ParseExecString(execString string) (string, []string) {
	execSplit := strings.Fields(execString)
	return execSplit[0], execSplit[1:]
}
