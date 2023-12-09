package composable

import (
	"context"
	"log"
	"os"
	"os/exec"
	"runtime"
)

// ProcessStatus represents states of a Process.
type ProcessStatus string

const (
	// ProcessNotStarted is the status of a Process before Process.Start called.
	ProcessNotStarted ProcessStatus = "NotStarted"
	// ProcessRunning is the status of a running Process.
	ProcessRunning ProcessStatus = "Running"
	// ProcessStopped is the status of a stopped Process.
	ProcessStopped ProcessStatus = "Stopped"
	// ProcessError is the status of a Process that has failed to start or exited with an error exit code.
	ProcessError ProcessStatus = "Error"
)

// Process is a state machine and container for exec.Cmd, maintaining status with ProcessStatus.
type Process struct {
	Binary         string               `json:"binary"`
	Args           []string             `json:"args"`
	Dir            string               `json:"dir"`
	ProcessStatus  ProcessStatus        `json:"status"`
	ProcessStatusC <-chan ProcessStatus `json:"-"`
	Command        *exec.Cmd            `json:"-"`
	processStatusC chan<- ProcessStatus
	termFunc       func()
}

// NewProcess creates a Process for a given binary with arguments and a work directory.
func NewProcess(binary string, args []string, dir string) *Process {
	c := make(chan ProcessStatus)
	return &Process{
		Binary:         binary,
		Args:           args,
		Dir:            dir,
		ProcessStatus:  ProcessNotStarted,
		ProcessStatusC: c,
		processStatusC: c,
	}
}

// Restart conditionally calls Process.Stop if Process is running before calling Process.Start.
func (p *Process) Restart() {
	log.Println("[DEBUG] Process.Restart", p.Binary)
	if p.Command != nil && p.Command.ProcessState != nil && !p.Command.ProcessState.Exited() {
		p.Stop()
	}
	p.Start()
}

// Start initializes Process.Command with an exec.Cmd and starts the process with Run().
// This func will block the calling goroutine.
func (p *Process) Start() {
	log.Println("[DEBUG] Process.Start", p.Binary)
	ctx, cancelFunc := context.WithCancel(context.Background())
	p.termFunc = cancelFunc
	p.Command = exec.CommandContext(ctx, p.Binary, p.Args...)
	p.Command.Stdout = os.Stdout
	p.Command.Stderr = os.Stderr
	p.Command.Dir = p.Dir
	p.updateStatus(ProcessRunning)
	err := p.Command.Run()
	if err != nil && !p.isCancelledCmdError(err) {
		log.Println("[ERROR] Process.Start error", err.Error())
		p.updateStatus(ProcessError)
	} else {
		log.Println("[DEBUG] Process.Start stopped")
		p.updateStatus(ProcessStopped)
	}
}

func (p *Process) Status() CompositionStatus {
	switch p.ProcessStatus {
	case ProcessNotStarted:
		return CompositionNotStarted
	case ProcessStopped:
		return CompositionStopped
	case ProcessError:
		return CompositionError
	case ProcessRunning:
		return CompositionRunning
	}
	return CompositionNotStarted
}

// Stop uses a context.CancelFunc created for exec.CommandContext to stop its running process.
func (p *Process) Stop() {
	log.Println("[DEBUG] Process.Stop", p.Binary)
	if p.termFunc != nil {
		p.termFunc()
		p.termFunc = nil
	}
	p.updateStatus(ProcessStopped)
}

// updateStatus pushes the new ProcessStatus to Process.StatusC readers.
func (p *Process) updateStatus(status ProcessStatus) {
	log.Println("[DEBUG] Process.updateStatus", p.ProcessStatus, "to", status)
	p.ProcessStatus = status
	select {
	case p.processStatusC <- status:
	default:
	}
}

// isCancelledCmdError determines whether an error from exec.Cmd is a killed signal message.
func (p *Process) isCancelledCmdError(err error) bool {
	if p.ProcessStatus == ProcessStopped {
		switch runtime.GOOS {
		case "windows":
			return true
		default:
			return err.Error() == "signal: killed"
		}
	}
	return false
}
