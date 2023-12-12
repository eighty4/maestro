package composable

import (
	"context"
	"log"
	"os"
	"os/exec"
	"runtime"
)

// Process is a state machine and container for exec.Cmd, maintaining status with ProcessStatus.
type Process struct {
	Binary        string    `json:"binary"`
	Args          []string  `json:"args"`
	Dir           string    `json:"dir"`
	CurrentStatus Status    `json:"status"`
	Command       *exec.Cmd `json:"-"`
	statusC       chan Status
	termFunc      func()
}

// NewProcess creates a Process for a given binary with arguments and a work directory.
func NewProcess(binary string, args []string, dir string) *Process {
	return &Process{
		Binary:  binary,
		Args:    args,
		Dir:     dir,
		statusC: make(chan Status),
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
	p.updateStatus(Running)
	err := p.Command.Run()
	if err != nil && !p.isCancelledCmdError(err) {
		log.Println("[ERROR] Process.Start error", err.Error())
		p.updateStatus(Error)
	} else {
		log.Println("[DEBUG] Process.Start stopped")
		p.updateStatus(Stopped)
	}
}

func (p *Process) Status() Status {
	return p.CurrentStatus
}

func (p *Process) StatusC() <-chan Status {
	return p.statusC
}

// Stop uses a context.CancelFunc created for exec.CommandContext to stop its running process.
func (p *Process) Stop() {
	log.Println("[DEBUG] Process.Stop", p.Binary)
	if p.termFunc != nil {
		p.termFunc()
		p.termFunc = nil
	}
	p.updateStatus(Stopped)
}

// updateStatus pushes the new ProcessStatus to Process.StatusC readers.
func (p *Process) updateStatus(status Status) {
	log.Println("[DEBUG] Process.updateStatus", p.CurrentStatus, "to", status)
	p.CurrentStatus = status
	select {
	case p.statusC <- p.Status():
	default:
	}
}

// isCancelledCmdError determines whether an error from exec.Cmd is a killed signal message.
func (p *Process) isCancelledCmdError(err error) bool {
	if p.CurrentStatus == Stopped {
		switch runtime.GOOS {
		case "windows":
			return true
		default:
			return err.Error() == "signal: killed"
		}
	}
	return false
}
