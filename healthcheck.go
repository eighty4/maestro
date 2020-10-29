package main

import (
	"time"
)

type HealthcheckStatus string

const (
	HealthcheckPending = "Pending"
	HealthcheckPassing = "Healthy"
	HealthcheckFailing = "Failing"
)

type Healthcheck struct {
	Config       *HealthcheckConfig
	Process      *Process
	Ticker       *time.Ticker
	Status       HealthcheckStatus
	StatusUpdate chan HealthcheckStatus
}

func NewHealthcheck(config *HealthcheckConfig, context *MaestroContext) *Healthcheck {
	return &Healthcheck{
		Config:       config,
		Process:      NewProcessFromExecString(config.Cmd, context.WorkDir),
		Ticker:       time.NewTicker(time.Second * time.Duration(config.Interval)),
		Status:       HealthcheckPending,
		StatusUpdate: make(chan HealthcheckStatus),
	}
}

func (hc *Healthcheck) Start() {
	if hc.Config.Delay > 0 {
		<-time.NewTimer(time.Second * time.Duration(hc.Config.Delay)).C
	}
	for {
		go hc.Process.Restart()
		done := false
		for !done {
			switch <-hc.Process.StatusUpdate {
			case ProcessStopped:
				if hc.Process.Command.ProcessState.ExitCode() > 0 {
					hc.updateStatus(HealthcheckFailing)
				} else {
					hc.updateStatus(HealthcheckPassing)
				}
				done = true
				break
			case ProcessError:
				hc.updateStatus(HealthcheckFailing)
				done = true
				break
			}
		}
		if hc.Ticker != nil {
			<-hc.Ticker.C
		}
	}
}

func (hc *Healthcheck) updateStatus(status HealthcheckStatus) {
	hc.Status = status
	select {
	case hc.StatusUpdate <- status:
	default:
	}
}
