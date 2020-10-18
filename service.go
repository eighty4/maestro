package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func LaunchService(config ServiceConfig) {
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
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("started service")

	// todo start healthcheck timer
}
