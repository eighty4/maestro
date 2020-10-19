package main

import (
	"fmt"
	"github.com/otaviokr/topological-sort/toposort"
	"log"
	"os"
	"os/exec"
	"strings"
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
	var command *exec.Cmd
	if config.Gradle != nil {
		command = exec.Command("./gradlew", "-q", "--console=plain", fmt.Sprintf("%s:%s", config.Gradle.Module, config.Gradle.Task))
	} else {
		log.Fatalln("invalid service config?")
	}

	// todo color-coded service name appended to each log line
	command.Stdout = os.Stdout
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	go func() {
		err := command.Run()
		if err != nil {
			log.Fatal(err)
		}
		// todo start healthcheck timer
	}()
}
