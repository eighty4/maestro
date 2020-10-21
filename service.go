package main

import (
	"fmt"
	"github.com/otaviokr/topological-sort/toposort"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func InitServices(config *Config) {
	startOrder, err := sortServiceDeps(config.Services)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("service start order: %v\n", startOrder)
	for _, dep := range startOrder {
		LaunchService(config.ServicesByName[dep])
	}
}

func IsCircularDepError(err error) bool {
	return strings.HasPrefix(err.Error(), "circular dep found for service")
}

func sortServiceDeps(services []*ServiceConfig) ([]string, error) {
	depTree := map[string][]string{}
	for _, serviceConfig := range services {
		depTree[serviceConfig.Name] = serviceConfig.DependsOn
	}
	sortedDeps, err := toposort.ReverseTarjan(depTree)
	if err != nil {
		if strings.HasPrefix(err.Error(), "Found cycle at node") {
			return nil, fmt.Errorf("circular dep found for service " + strings.Split(err.Error(), ":")[1][1:])
		} else {
			return nil, err
		}
	}
	return sortedDeps, nil
}

func LaunchService(config *ServiceConfig) {
	log.Println("starting service " + config.Name)
	var sCmd *exec.Cmd
	if len(config.Exec) > 0 {
		sCmd = createExecCmd(config.Exec)
	} else if config.Gradle != nil {
		sCmd = exec.Command("./gradlew", "-q", "--console=plain", fmt.Sprintf("%s:%s", config.Gradle.Module, config.Gradle.Task))
	} else {
		log.Fatalln("invalid service config?")
	}

	// todo color-coded service name appended to each log line
	sCmd.Stdout = os.Stdout
	sCmd.Stderr = os.Stderr

	var hcTicker *time.Ticker
	go func() {
		err := sCmd.Run()
		if err != nil {
			log.Println(config.Name, err.Error())
		}
		if hcTicker != nil {
			hcTicker.Stop()
			hcTicker = nil
		}
	}()

	if config.Healthcheck != nil {
		go func() {
			if config.Healthcheck.Delay > 0 {
				<-time.NewTimer(time.Second * time.Duration(config.Healthcheck.Delay)).C
			}
			hcTicker = time.NewTicker(time.Second * time.Duration(config.Healthcheck.Interval))
			for {
				if hcTicker == nil {
					return
				}

				hcCmd := createExecCmd(config.Healthcheck.Cmd)
				err := hcCmd.Run()
				if err != nil {
					log.Printf("%s hc cmd (%s) err: %s\n", config.Name, config.Healthcheck.Cmd, err.Error())
				} else if !hcCmd.ProcessState.Success() {
					log.Printf("%s hc cmd (%s) err: %d\n", config.Name, config.Healthcheck.Cmd, hcCmd.ProcessState.ExitCode())
				} else {
					log.Printf("%s hc: healthy\n", config.Name)
				}
				<-hcTicker.C
			}
		}()
	}
}

func createExecCmd(execString string) *exec.Cmd {
	binary, args := parseExecString(execString)
	return exec.Command(binary, args...)
}

func parseExecString(execString string) (string, []string) {
	execSplit := strings.Fields(execString)
	return execSplit[0], execSplit[1:]
}
