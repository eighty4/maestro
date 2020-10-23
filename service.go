package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

type MaestroContext struct {
	WorkDir string
}

type ServiceStatus uint8

const (
	Starting = iota // initial status before run or healthcheck
	Running         // running service w/o healthcheck
	Healthy         // running service with passing healthcheck
	Failing         // running service with failing healthcheck
	Stopped         // stopped service with exit code 0
	Error           // stopped service with non-zero exit code
)

func ServiceStatusString(status ServiceStatus) string {
	switch status {
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case Healthy:
		return "Healthy"
	case Failing:
		return "Failing"
	case Stopped:
		return "Stopped"
	case Error:
		return "Error"
	default:
		return "Unknown"
	}
}

type ServiceProcess struct {
	Context           *MaestroContext
	Config            *ServiceConfig
	Command           *exec.Cmd
	HealthcheckTicker *time.Ticker
}

func NewServiceProcess(serviceConfig *ServiceConfig, context *MaestroContext) *ServiceProcess {
	return &ServiceProcess{
		Command: createServiceCommand(serviceConfig, context),
		Config:  serviceConfig,
		Context: context,
	}
}

func (sp *ServiceProcess) Launch() <-chan ServiceStatus {
	status := make(chan ServiceStatus)
	go func() {
		status <- Starting
		if sp.Config.Healthcheck == nil {
			go sp.runServiceProcess(status)
			status <- Running
		} else {
			go sp.runServiceProcess(status)
			go sp.runServiceHealthcheck(status)
		}
	}()
	return status
}

func (sp *ServiceProcess) runServiceProcess(status chan<- ServiceStatus) {
	log.Println(sp.Config.Name + " starting")
	err := sp.Command.Run()
	log.Printf("%s has exited with status %d\n", sp.Config.Name, sp.Command.ProcessState.ExitCode())
	if sp.HealthcheckTicker != nil {
		sp.HealthcheckTicker.Stop()
		sp.HealthcheckTicker = nil
	}
	if err != nil && sp.Command.ProcessState.ExitCode() == -1 {
		log.Fatalf("%s's cmd is mis-configured: %s\n", sp.Config.Name, err.Error())
	} else if sp.Command.ProcessState.ExitCode() > 0 {
		log.Printf("%s exited with status %d", sp.Config.Name, sp.Command.ProcessState.ExitCode())
		status <- Error
	} else {
		status <- Stopped
	}
}

func (sp *ServiceProcess) runServiceHealthcheck(status chan<- ServiceStatus) {
	if sp.Config.Healthcheck.Delay > 0 {
		<-time.NewTimer(time.Second * time.Duration(sp.Config.Healthcheck.Delay)).C
	}
	sp.HealthcheckTicker = time.NewTicker(time.Second * time.Duration(sp.Config.Healthcheck.Interval))
	for {
		if sp.HealthcheckTicker == nil {
			return
		}
		healthcheckCommand := createExecCmd(sp.Config.Healthcheck.Cmd)
		err := healthcheckCommand.Run()
		if err != nil && healthcheckCommand.ProcessState.ExitCode() == -1 {
			log.Fatalf("%s healthcheck cmd is mis-configured: %s\n", sp.Config.Name, err.Error())
		} else if healthcheckCommand.ProcessState.ExitCode() > 0 {
			//log.Println(sp.Config.Name + " hc failing")
			status <- Failing
		} else {
			//log.Println(sp.Config.Name + " hc healthy")
			status <- Healthy
		}
		if sp.HealthcheckTicker != nil {
			<-sp.HealthcheckTicker.C
		}
	}
}

func InitServices(config *MaestroConfig, context *MaestroContext) {
	pending := map[string][]string{}
	var ready []string
	services := map[string]*ServiceProcess{}
	for _, serviceConfig := range config.Services {
		serviceProcess := NewServiceProcess(serviceConfig, context)
		services[serviceConfig.Name] = serviceProcess
		if len(serviceConfig.DependsOn) == 0 {
			ready = append(ready, serviceConfig.Name)
		} else {
			pending[serviceConfig.Name] = append([]string(nil), serviceConfig.DependsOn...)
		}
	}

	var resolveDependency func(string) []string
	var launchService func(serviceName string)
	launchService = func(serviceName string) {
		status := services[serviceName].Launch()
		for {
			cur := <-status
			if cur == Running || cur == Healthy {
				resolvables := resolveDependency(serviceName)
				if len(resolvables) > 0 {
					for _, resolvable := range resolvables {
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

func createServiceCommand(config *ServiceConfig, context *MaestroContext) *exec.Cmd {
	var serviceCommand *exec.Cmd
	if len(config.Exec) > 0 {
		serviceCommand = createExecCmd(config.Exec)
	} else if config.Gradle != nil {
		serviceCommand = exec.Command("./gradlew", "-q", "--console=plain", fmt.Sprintf("%s:%s", config.Gradle.Module, config.Gradle.Task))
	} else if config.Npm != nil {
		npmRunCmdString := "npm run " + config.Npm.Script
		if len(config.Npm.Args) > 0 {
			npmRunCmdString += " -- " + config.Npm.Args
		}
		serviceCommand = createExecCmd(npmRunCmdString)
		if len(config.Npm.RelDir) > 0 {
			serviceCommand.Dir = path.Join(context.WorkDir, config.Npm.RelDir)
		}
	} else {
		log.Fatalln("invalid service config?")
	}

	// todo color-coded service name appended to each log line
	serviceCommand.Stdout = os.Stdout
	serviceCommand.Stderr = os.Stderr
	return serviceCommand
}

func createExecCmd(execString string) *exec.Cmd {
	binary, args := parseExecString(execString)
	return exec.Command(binary, args...)
}

func parseExecString(execString string) (string, []string) {
	execSplit := strings.Fields(execString)
	return execSplit[0], execSplit[1:]
}
