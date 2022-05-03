package main

import (
	"log"
	"time"
)

// todo buffer holding all log data from a service (GET /logs/${serviceName})
// todo non-blocking channel for new logs used by browser sse and log udp connections
// todo restart and stop process commands

func main() {
	context, err := NewMaestroContext()
	if err != nil {
		log.Fatalln(err)
	}
	switch context.Command.Op {
	case Main:
		if context.ConfigFile == nil {
			log.Println("no maestro config in this directory")
			// todo read build.gradle, package.json, docker-compose.yml for services to create
		} else {
			StartFrontend()
			<-time.NewTimer(100 * time.Millisecond).C
			InitServices(context)
			select {}
		}
		break
	case Logs:
		log.Println("log command is a noop")
		break
	case Git:
		NewWorkspaceGitPull(context).pull()
		break
	}
}
