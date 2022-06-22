package main

import (
	"log"
	"sync"
)

type ServiceStatus string

const (
	ServiceStarting = "Starting" // initial status before run or healthcheck
	ServiceRunning  = "Running"  // running service w/o healthcheck
	ServiceHealthy  = "Healthy"  // running service with passing healthcheck
	ServiceFailing  = "Failing"  // running service with failing healthcheck
	ServiceStopped  = "Stopped"  // stopped service with exit code 0
	ServiceError    = "Error"    // stopped service with non-zero exit code
)

type ManagedService struct {
	Context     *MaestroContext
	Config      *ServiceConfig
	Process     *Process
	Healthcheck *Healthcheck `json:"omitempty"`
	Status      ServiceStatus
}

func NewManagedService(serviceConfig *ServiceConfig, context *MaestroContext) *ManagedService {
	service := &ManagedService{
		Context: context,
		Config:  serviceConfig,
		Process: serviceConfig.ProcessConfig.CreateProcess(context),
		Status:  ServiceStopped,
	}
	service.Process.Logging.Prefix = serviceConfig.Name
	return service
}

func (ms *ManagedService) Launch() <-chan ServiceStatus {
	status := make(chan ServiceStatus)
	go ms.Process.Start()
	if ms.Config.Healthcheck != nil {
		ms.Healthcheck = NewHealthcheck(ms.Config.Healthcheck, ms.Context)
		go ms.Healthcheck.Start()
	}
	go ms.waitForServiceReady(status)
	return status
}

func (ms *ManagedService) waitForServiceReady(status chan<- ServiceStatus) {
	ms.updateStatus(status, ServiceStarting)
	var hcStatus <-chan HealthcheckStatus
	var pStatus <-chan ProcessStatus
	if ms.Healthcheck != nil {
		hcStatus = ms.Healthcheck.StatusUpdate
	} else {
		pStatus = ms.Process.StatusUpdate
	}
	for {
		select {
		case s := <-hcStatus:
			if s == HealthcheckPassing {
				ms.updateStatus(status, ServiceHealthy)
				return
			}
			break
		case s := <-pStatus:
			if s == ProcessRunning {
				ms.updateStatus(status, ServiceRunning)
				return
			}
			break
		}
	}
}

func (ms *ManagedService) updateStatus(c chan<- ServiceStatus, s ServiceStatus) {
	ms.Status = s
	select {
	case c <- s:
	default:
	}
}

var services = make(map[string]*ManagedService)

func InitServices(context *MaestroContext) {
	pending := map[string][]string{}
	var pendingNames []string
	var ready []string
	for _, serviceConfig := range context.Services {
		serviceProcess := NewManagedService(serviceConfig, context)
		services[serviceConfig.Name] = serviceProcess
		if len(serviceConfig.DependsOn) == 0 {
			ready = append(ready, serviceConfig.Name)
		} else {
			pending[serviceConfig.Name] = append([]string(nil), serviceConfig.DependsOn...)
			pendingNames = append(pendingNames, serviceConfig.Name)
		}
	}
	log.Println("starting services without dependencies", ready)
	log.Println("services pending dependencies", pendingNames)

	var resolveDependency func(string) []string
	var launchService func(serviceName string)
	launchService = func(serviceName string) {
		status := services[serviceName].Launch()
		for {
			if next := <-status; next == ServiceRunning || next == ServiceHealthy {
				resolvables := resolveDependency(serviceName)
				if len(resolvables) > 0 {
					for _, resolvable := range resolvables {
						log.Println("starting service", resolvable)
						go launchService(resolvable)
					}
				}
			}
		}
	}

	if len(pending) == 0 {
		resolveDependency = func(ignore string) []string { return nil }
	} else {
		mutex := sync.Mutex{}
		resolveDependency = func(resolved string) []string {
			mutex.Lock()
			updates := map[string][]string{}
			for serviceName, deps := range pending {
				for i, dep := range deps {
					if dep == resolved {
						s := append([]string(nil), deps...)
						s[len(s)-1], s[i] = s[i], s[len(s)-1]
						updates[serviceName] = s[:len(s)-1]
					}
				}
			}
			var resolvable []string
			for serviceName, deps := range updates {
				if len(deps) == 0 {
					resolvable = append(resolvable, serviceName)
					delete(pending, serviceName)
				} else {
					pending[serviceName] = deps
				}
			}
			mutex.Unlock()
			return resolvable
		}
	}

	for _, serviceName := range ready {
		go launchService(serviceName)
	}
}
