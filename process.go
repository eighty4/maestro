package main

import (
	"context"
	"log"
	"os/exec"
	"strings"
)

type ProcessStatus string

const (
	ProcessRunning = "Running" // running service
	ProcessStopped = "Stopped" // stopped service with exit code 0
	ProcessError   = "Error"   // stopped service with non-zero exit code
)

type Process struct {
	Binary       string
	Args         []string
	Dir          string
	Env          *map[string]string `json:",omitempty"`
	Logging      *Logger            `json:"-"`
	Status       ProcessStatus
	StatusUpdate chan ProcessStatus `json:"-"`
	Command      *exec.Cmd          `json:"-"`
	termFunc     func()
}

func NewProcess(binary string, args []string, dir string) *Process {
	return &Process{
		Binary:       binary,
		Args:         args,
		Dir:          dir,
		Logging:      NewProcessLogger(false),
		Status:       ProcessStopped,
		StatusUpdate: make(chan ProcessStatus),
	}
}

func NewProcessFromExecString(execString string, dir string) *Process {
	binary, args := ParseExecString(execString)
	return NewProcess(binary, args, dir)
}

func (p *Process) Start() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	p.termFunc = cancelFunc
	p.Command = exec.CommandContext(ctx, p.Binary, p.Args...)
	p.Command.Stdout = p.Logging
	p.Command.Stderr = p.Logging
	p.Command.Dir = p.Dir
	p.updateStatus(ProcessRunning)
	err := p.Command.Run()
	if err != nil {
		if err.Error() == "signal: killed" {
			p.updateStatus(ProcessStopped)
		} else if p.Command != nil && p.Command.ProcessState != nil && p.Command.ProcessState.ExitCode() == -1 {
			log.Fatalf("cmd is mis-configured: %s\n", err.Error())
		} else {
			p.updateStatus(ProcessError)
		}
	} else {
		p.updateStatus(ProcessStopped)
	}
}

func (p *Process) Restart() {
	if p.Command != nil && p.Command.ProcessState != nil && !p.Command.ProcessState.Exited() {
		p.Stop()
	}
	p.Start()
}

func (p *Process) updateStatus(status ProcessStatus) {
	p.Status = status
	select {
	case p.StatusUpdate <- status:
	default:
	}
}

func (p *Process) Stop() {
	if p.termFunc != nil {
		p.termFunc()
		p.termFunc = nil
	}
	p.Command = nil
	p.updateStatus(ProcessStopped)
}

func ParseExecString(execString string) (string, []string) {
	execSplit := strings.Fields(execString)
	return execSplit[0], execSplit[1:]
}
