package main

import (
	"github.com/eighty4/maestro/composable"
	"github.com/eighty4/maestro/util"
	"log"
	"sync"
)

type ServiceStatus string

const (
	ServiceStarting ServiceStatus = "Starting" // initial status before run or healthcheck
	ServiceRunning  ServiceStatus = "Running"  // running service w/o healthcheck
	ServiceHealthy  ServiceStatus = "Healthy"  // running service with passing healthcheck
	ServiceFailing  ServiceStatus = "Failing"  // running service with failing healthcheck
	ServiceStopped  ServiceStatus = "Stopped"  // stopped service with exit code 0
	ServiceError    ServiceStatus = "Error"    // stopped service with non-zero exit code
)

type ManagedService struct {
	Context     *MaestroContext
	Config      *ServiceConfig
	Process     *composable.Process
	Healthcheck *composable.Healthcheck `json:"omitempty"`
	Status      ServiceStatus
}

func NewManagedService(serviceConfig *ServiceConfig, context *MaestroContext) *ManagedService {
	service := &ManagedService{
		Context: context,
		Config:  serviceConfig,
		Process: serviceConfig.ProcessConfig.CreateProcess(context),
		Status:  ServiceStopped,
	}
	return service
}

func (ms *ManagedService) Launch() <-chan ServiceStatus {
	status := make(chan ServiceStatus)
	go ms.Process.Start()
	if ms.Config.Healthcheck != nil {
		hc := ms.Config.Healthcheck
		exec := composable.ParseCmdString(hc.Cmd, ms.Context.WorkDir)
		ms.Healthcheck = composable.NewExecHealthcheck(exec, util.Seconds(hc.Delay), util.Seconds(hc.Interval))
		go ms.Healthcheck.Start()
	}
	go ms.waitForServiceReady(status)
	return status
}

func (ms *ManagedService) waitForServiceReady(status chan<- ServiceStatus) {
	print("waitForServiceReady\n")
	ms.updateStatus(status, ServiceStarting)
	var hcStatus <-chan composable.HealthcheckStatus
	var pStatus <-chan composable.ProcessStatus
	if ms.Healthcheck != nil {
		hcStatus = ms.Healthcheck.StatusC
	} else {
		pStatus = ms.Process.StatusC
	}
	for {
		select {
		case s := <-hcStatus:
			if s == composable.HealthcheckPassing {
				ms.updateStatus(status, ServiceHealthy)
				return
			}
			break
		case s := <-pStatus:
			if s == composable.ProcessRunning {
				ms.updateStatus(status, ServiceRunning)
				return
			}
			break
		}
	}
}

func (ms *ManagedService) updateStatus(c chan<- ServiceStatus, s ServiceStatus) {
	log.Println("[DEBUG] ManagedService.updateStatus", s)
	ms.Status = s
	select {
	case c <- s:
	default:
	}
}

type ServiceOrchestration struct {
	context  *MaestroContext
	mutex    sync.Mutex
	pending  map[string][]string
	Services map[string]*ManagedService
}

func NewServiceOrchestration(context *MaestroContext) *ServiceOrchestration {
	return &ServiceOrchestration{
		context:  context,
		mutex:    sync.Mutex{},
		pending:  make(map[string][]string),
		Services: make(map[string]*ManagedService),
	}
}

func (o *ServiceOrchestration) Initialize() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	pending := map[string][]string{}
	var pendingNames []string
	var ready []string
	for _, serviceConfig := range o.context.Services {
		serviceProcess := NewManagedService(serviceConfig, o.context)
		o.Services[serviceConfig.Name] = serviceProcess
		if len(serviceConfig.DependsOn) == 0 {
			ready = append(ready, serviceConfig.Name)
		} else {
			pending[serviceConfig.Name] = append([]string(nil), serviceConfig.DependsOn...)
			pendingNames = append(pendingNames, serviceConfig.Name)
		}
	}
	log.Println("starting services without dependencies", ready)
	log.Println("services pending dependencies", pendingNames)

	for _, serviceName := range ready {
		go o.LaunchService(serviceName)
	}
}

func (o *ServiceOrchestration) LaunchService(serviceName string) {
	status := o.Services[serviceName].Launch()
	for {
		if next := <-status; next == ServiceRunning || next == ServiceHealthy {
			resolvables := o.ResolveDependency(serviceName)
			if len(resolvables) > 0 {
				for _, resolvable := range resolvables {
					log.Println("starting service", resolvable)
					go o.LaunchService(resolvable)
				}
			}
		}
	}
}

func (o *ServiceOrchestration) ResolveDependency(resServiceName string) []string {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	updates := map[string][]string{}
	for serviceName, deps := range o.pending {
		for i, depServiceName := range deps {
			if depServiceName == resServiceName {
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
			delete(o.pending, serviceName)
		} else {
			o.pending[serviceName] = deps
		}
	}
	return resolvable
}
