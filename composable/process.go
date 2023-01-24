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
	Binary   string               `json:"binary"`
	Args     []string             `json:"args"`
	Dir      string               `json:"dir"`
	Status   ProcessStatus        `json:"status"`
	StatusC  <-chan ProcessStatus `json:"-"`
	Command  *exec.Cmd            `json:"-"`
	statusC  chan<- ProcessStatus
	termFunc func()
}

// NewProcess creates a Process for a given binary with arguments and a work directory.
func NewProcess(binary string, args []string, dir string) *Process {
	c := make(chan ProcessStatus)
	return &Process{
		Binary:  binary,
		Args:    args,
		Dir:     dir,
		Status:  ProcessNotStarted,
		StatusC: c,
		statusC: c,
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
	log.Println("[DEBUG] Process.updateStatus", p.Status, "to", status)
	p.Status = status
	select {
	case p.statusC <- status:
	default:
	}
}

// isCancelledCmdError determines whether an error from exec.Cmd is a killed signal message.
func (p *Process) isCancelledCmdError(err error) bool {
	if p.Status == ProcessStopped {
		switch runtime.GOOS {
		case "windows":
			log.Fatalln("[ERROR] Process.isCancelledCmdError unsupported on Windows")
		default:
			return err.Error() == "signal: killed"
		}
	}
	return false
}
