package composable

import (
	"context"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

// Process is a state machine and container for exec.Cmd, maintaining status with Status.
type Process struct {
	Binary        string    `json:"binary"`
	Args          []string  `json:"args"`
	Dir           string    `json:"dir"`
	CurrentStatus Status    `json:"status"`
	Command       *exec.Cmd `json:"-"`
	statusC       chan Status
	termFunc      func()
	stoppedC      chan int
	mutex         sync.Mutex
}

// NewProcess creates a Process for a given binary with arguments and a work directory.
func NewProcess(binary string, args []string, dir string) *Process {
	return &Process{
		Binary:        binary,
		Args:          args,
		Dir:           dir,
		CurrentStatus: NotStarted,
		statusC:       make(chan Status),
		stoppedC:      make(chan int),
	}
}

// Restart conditionally calls Process.Stop if Process is running before calling Process.Start.
func (p *Process) Restart() {
	log.Println("[DEBUG] Process.Restart", p.Binary)
	if p.Command != nil && p.Command.Process != nil {
		p.Stop()
	}
	p.Start()
}

// Start initializes Process.Command with an exec.Cmd and starts the process with Run().
// This func will block the calling goroutine.
func (p *Process) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	log.Println("[DEBUG] Process.Start", p.Binary)
	ctx, cancelFunc := context.WithCancel(context.Background())
	p.termFunc = cancelFunc
	p.Command = exec.CommandContext(ctx, p.Binary, p.Args...)
	p.Command.Stdout = os.Stdout
	p.Command.Stderr = os.Stderr
	p.Command.Dir = p.Dir
	log.Println("[DEBUG] Process.Start starting")
	p.updateStatus(Starting)
	err := p.Command.Start()
	if err != nil && !p.isCancelledCmdError(err) {
		log.Println("[ERROR] Process.Start error", err.Error())
		p.updateStatus(Error)
	} else {
		log.Println("[DEBUG] Process.Start running")
		p.updateStatus(Running)
		go func() {
			cmd := p.Command
			err = cmd.Wait()
			if p.Command != cmd {
				return
			}
			var resultStatus Status
			if err != nil {
				if p.isCancelledCmdError(err) {
					log.Printf("[DEBUG] Process.Start error on command wait return determined to be cancel: %s\n", err)
					resultStatus = Stopped
				} else {
					log.Printf("[DEBUG] Process.Start error on command wait return: %s\n", err)
					resultStatus = Error
				}
			} else {
				log.Println("[DEBUG] Process.Start after command wait return without error")
				resultStatus = Stopped
			}
			if resultStatus != p.CurrentStatus {
				p.updateStatus(resultStatus)
				select {
				case p.stoppedC <- cmd.Process.Pid:
				default:
				}
			}
		}()
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
	p.mutex.Lock()
	defer p.mutex.Unlock()
	log.Println("[DEBUG] Process.Stop", p.Binary)
	if p.Command != nil && p.Command.Process != nil && p.Command.ProcessState == nil {
		pid := p.Command.Process.Pid
		p.termFunc()
		p.termFunc = nil
		if pid != <-p.stoppedC {
			log.Fatalln("a most unlikely event")
		}
	}
}

// updateStatus pushes the new Status to Process.StatusC readers.
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
	switch runtime.GOOS {
	case "windows":
		return true
	default:
		return err.Error() == "signal: killed"
	}
}
