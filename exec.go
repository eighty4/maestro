package main

import (
	"strings"
)

type ExecConfig struct {
	Cmd string
}

func (c *ExecConfig) CreateProcess(context *MaestroContext) *Process {
	binary, args := ParseExecString(c.Cmd)
	process := NewProcess(binary, args, context.WorkDir)
	process.Logging.print = true
	return process
}

func ParseExecString(execString string) (string, []string) {
	execSplit := strings.Fields(execString)
	return execSplit[0], execSplit[1:]
}
