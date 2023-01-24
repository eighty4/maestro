package composable

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// HealthcheckStatus represents states of a Healthcheck.
type HealthcheckStatus string

const (
	HealthcheckPending HealthcheckStatus = "Pending"
	HealthcheckPassing HealthcheckStatus = "Passing"
	HealthcheckFailing HealthcheckStatus = "Failing"
	HealthcheckStopped HealthcheckStatus = "Stopped"
)

// HealthcheckMethod provides Healthcheck API with an abstraction for performing different types of health checks.
type HealthcheckMethod interface {
	// Perform is called at each Healthcheck.Interval with a sending bool chan to report result.
	Perform(result chan<- bool)
	// Destroy will be called when a Healthcheck is finished.
	// Destroy implementations should close and clean up open resources.
	Destroy()
}

// Healthcheck manages a time.Ticker and HealthcheckMethod to perform health checks.
// The frequency of health checking is controlled by Healthcheck.Interval.
// Use Healthcheck.Delay to initially pause health checking for a system under test to complete initialization.
type Healthcheck struct {
	Delay    time.Duration
	Interval time.Duration
	Status   HealthcheckStatus
	StatusC  <-chan HealthcheckStatus `json:"-"`
	method   HealthcheckMethod
	running  atomic.Bool
	statusC  chan<- HealthcheckStatus
	ticker   *time.Ticker
}

// NewHealthcheck creates a new Healthcheck for a given HealthcheckMethod to run at an interval specified by time.Duration.
// An optional time.Duration for Healthcheck.Delay allows a system under test to be given sufficient startup time.
func NewHealthcheck(method HealthcheckMethod, delay time.Duration, interval time.Duration) *Healthcheck {
	statusC := make(chan HealthcheckStatus)
	return &Healthcheck{
		Delay:    delay,
		Interval: interval,
		Status:   HealthcheckPending,
		StatusC:  statusC,
		method:   method,
		statusC:  statusC,
	}
}

// Start creates a time.Ticker to trigger health checks and continues calling HealthcheckMethod.Perform until Healthcheck.Stop is called.
func (hc *Healthcheck) Start() {
	hc.running.Swap(true)
	if hc.Delay > 0 {
		log.Println("[DEBUG] Healthcheck.Start delay", hc.Delay)
		<-time.NewTimer(hc.Delay).C
	}
	log.Println("[DEBUG] Healthcheck.Start interval", hc.Interval)
	hc.ticker = time.NewTicker(hc.Interval)
	hcResultC := make(chan bool)
	for {
		if !hc.running.Load() {
			return
		}
		select {
		case <-hc.ticker.C:
			log.Println("[DEBUG] Healthcheck.Start tick")
			go hc.method.Perform(hcResultC)
		case passing := <-hcResultC:
			if hc.running.Load() {
				if passing {
					hc.updateStatus(HealthcheckPassing)
				} else {
					hc.updateStatus(HealthcheckFailing)
				}
			}
		}
	}
}

// Stop stops the time.Ticker that triggers health checks and delegates clean up to HealthcheckMethod.Destroy.
func (hc *Healthcheck) Stop() {
	log.Println("[DEBUG] Healthcheck.stop")
	hc.running.Swap(false)
	hc.ticker.Stop()
	hc.method.Destroy()
	hc.updateStatus(HealthcheckStopped)
}

// updateStatus pushes the new HealthcheckStatus to Healthcheck.StatusC readers.
func (hc *Healthcheck) updateStatus(status HealthcheckStatus) {
	log.Println("[DEBUG] Healthcheck.updateStatus", hc.Status, "to", status)
	hc.Status = status
	select {
	case hc.statusC <- status:
	default:
	}
}

// NewExecHealthcheck creates a Healthcheck from an ExecDescription using a ExecHealthcheckMethod.
func NewExecHealthcheck(exec *ExecDescription, delay time.Duration, interval time.Duration) *Healthcheck {
	p := exec.Process()
	m := &ExecHealthcheckMethod{p}
	return NewHealthcheck(m, delay, interval)
}

// ExecHealthcheckMethod is a HealthcheckMethod that runs a Process at every Healthcheck.Interval.
type ExecHealthcheckMethod struct {
	process *Process
}

func (hm *ExecHealthcheckMethod) Perform(hcResultC chan<- bool) {
	hm.process.Restart()
	for {
		switch <-hm.process.StatusC {
		case ProcessStopped:
			if hm.process.Command.ProcessState.ExitCode() > 0 {
				hcResultC <- false
			} else {
				hcResultC <- true
			}
			break
		case ProcessError:
			hcResultC <- false
			break
		}
	}
}

func (hm *ExecHealthcheckMethod) Destroy() {
	hm.process.Stop()
}

// NewHttpHealthcheck creates a Healthcheck from an HttpGetDescription.
func NewHttpHealthcheck(get *HttpGetDescription, delay time.Duration, interval time.Duration) *Healthcheck {
	url := get.Url()
	m := &HttpHealthcheckMethod{url}
	return NewHealthcheck(m, delay, interval)
}

// HttpHealthcheckMethod is a HealthcheckMethod that performs an HTTP GET request at every Healthcheck.Interval.
type HttpHealthcheckMethod struct {
	url string
}

func (hm *HttpHealthcheckMethod) Perform(hcResultC chan<- bool) {
	res, err := http.Get(hm.url)
	if err != nil {
		log.Println("[ERROR] HttpHealthcheckMethod.Perform", err)
		hcResultC <- false
	} else if res.StatusCode >= 300 {
		log.Println("[WARN] HttpHealthcheckMethod.Perform", res.StatusCode)
		hcResultC <- false
	} else {
		log.Println("[DEBUG] HttpHealthcheckMethod.Perform", res.StatusCode)
		hcResultC <- true
	}
}

func (hm *HttpHealthcheckMethod) Destroy() {
}
