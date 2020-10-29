package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type ProcessStatus string

const (
	ProcessStarting = "Starting" // initial status
	ProcessRunning  = "Running"  // running service
	ProcessStopped  = "Stopped"  // stopped service with exit code 0
	ProcessError    = "Error"    // stopped service with non-zero exit code
)

func ProcessStatusString(status ProcessStatus) string {
	switch status {
	case ProcessStarting:
		return "Starting"
	case ProcessRunning:
		return "Running"
	case ProcessStopped:
		return "Stopped"
	case ProcessError:
		return "Error"
	default:
		return "Unknown"
	}
}

type Process struct {
	Binary       string
	Args         []string
	Dir          string
	Env          *map[string]string
	Logging      *Logger `json:"-"`
	Status       ProcessStatus
	StatusUpdate chan ProcessStatus `json:"-"`
	Command      *exec.Cmd          `json:"-"`
}

func NewProcess(binary string, args []string, dir string) *Process {
	return &Process{
		Binary:       binary,
		Args:         args,
		Dir:          dir,
		Status:       ProcessStopped,
		StatusUpdate: make(chan ProcessStatus),
	}
}

func NewProcessFromExecString(execString string, dir string) *Process {
	binary, args := ParseExecString(execString)
	return NewProcess(binary, args, dir)
}

func (p *Process) Start() {
	p.Command = exec.Command(p.Binary, p.Args...)
	p.Command.Dir = p.Dir
	p.Logging = NewProcessLogger(p.Command)
	p.updateStatus(ProcessRunning)
	err := p.Command.Run()
	if err != nil && p.Command.ProcessState.ExitCode() == -1 {
		log.Fatalf("cmd is mis-configured: %s\n", err.Error())
	} else if p.Command.ProcessState.ExitCode() > 0 {
		//log.Printf("%s exited with status %d", p.Binary, p.Command.ProcessState.ExitCode())
		p.updateStatus(ProcessError)
	} else {
		p.updateStatus(ProcessStopped)
	}
}

func (p *Process) Restart() {
	if p.Command != nil && p.Command.ProcessState != nil && !p.Command.ProcessState.Exited() {
		log.Fatalln("restarting running process")
	}
	p.Start()
}

func (p *Process) updateStatus(status ProcessStatus) {
	p.Status = status
	select {
	case p.StatusUpdate <- status:
		fmt.Println("sending status " + ProcessStatusString(status))
	default:
	}
}

func ParseExecString(execString string) (string, []string) {
	execSplit := strings.Fields(execString)
	return execSplit[0], execSplit[1:]
}
