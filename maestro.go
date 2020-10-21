package main

import (
	"log"
	"os"
)

func main() {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal("could not get cwd", err)
	}
	config, err := ReadConfig(workDir)
	if err != nil {
		log.Fatalf("could not read config: %s", err)
	}

	if config == nil {
		// todo read build.gradle, package.json, docker-compose.yml for services to create
	} else {
		InitServices(config, &MaestroContext{workDir})
		select {}
	}
}
