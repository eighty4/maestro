package main

import (
	"log"
	"os"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("could not get cwd", err)
	}
	config, err := ReadConfig(wd)
	if err != nil {
		log.Fatalf("could not read config: %s", err)
	}

	if config == nil {
		// todo read build.gradle, package.json, docker-compose.yml for services to create
	} else {
		// todo launch services in dep hierarchy order
		for _, service := range config.Services {
			LaunchService(*service)
		}
		log.Printf("%d services started\n", len(config.Services))
	}
}
