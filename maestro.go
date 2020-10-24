package main

import (
	"log"
)

func main() {
	context, err := NewMaestroContext()
	if err != nil {
		log.Fatalln(err)
	}

	if context.ConfigFile == nil {
		// todo read build.gradle, package.json, docker-compose.yml for services to create
	} else {
		InitServices(context)
		select {}
	}
}
