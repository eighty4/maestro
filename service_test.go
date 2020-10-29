package main

import (
	"testing"
	"time"
)

func TestCreateServiceCommand_ForExecutableCommand(t *testing.T) {
	config := &ServiceConfig{
		Exec: "ls /",
	}
	context := &MaestroContext{WorkDir: tempDir()}
	defer cleanup(context.WorkDir)

	command := NewServiceProcess(config, context)
	if command.Binary != "ls" {
		t.Error(command.Binary)
	}
	if len(command.Args) != 1 || command.Args[0] != "/" {
		t.Error(command.Args)
	}
}

func TestCreateServiceCommand_ForGradleTask(t *testing.T) {
	config := &ServiceConfig{
		Gradle: &GradleTaskConfig{"my-module", "my-task"},
	}
	context := &MaestroContext{WorkDir: tempDir()}
	defer cleanup(context.WorkDir)

	command := NewServiceProcess(config, context)
	if command.Binary != "./gradlew" {
		t.Error(command.Binary)
	}
	if len(command.Args) != 3 || command.Args[0] != "-q" || command.Args[1] != "--console=plain" || command.Args[2] != "my-module:my-task" {
		t.Error(command.Args)
	}
}

func TestCreateServiceCommand_ForNpmScript(t *testing.T) {
	config := &ServiceConfig{
		Npm: &NpmScriptConfig{"start", "--foo=bar", "my-yarn-workspace"},
	}
	context := &MaestroContext{WorkDir: tempDir()}
	defer cleanup(context.WorkDir)

	command := NewServiceProcess(config, context)
	if command.Binary != "npm" {
		t.Error(command.Binary)
	}
	if len(command.Args) != 4 || command.Args[0] != "run" || command.Args[1] != "start" || command.Args[2] != "--" || command.Args[3] != "--foo=bar" {
		t.Error(command.Args)
	}
	if command.Dir != context.WorkDir+"/my-yarn-workspace" {
		t.Error(command.Dir)
	}
}

func TestManagedService_WithoutHealthcheck_EmitsRunning(t *testing.T) {
	config := &ServiceConfig{
		Exec: "sleep 2",
	}
	context := &MaestroContext{WorkDir: tempDir()}
	defer cleanup(context.WorkDir)

	service := NewManagedService(config, context)
	status := service.Launch()

	if next := <-status; next != ServiceStarting {
		t.Error(next)
	}
	if next := <-status; next != ServiceRunning {
		t.Error(next)
	}
}

func TestManagedService_WithHealthcheck_EmitsHealthy(t *testing.T) {
	config := &ServiceConfig{
		Exec: "sleep 2",
		Healthcheck: &HealthcheckConfig{
			Cmd:      "ls /",
			Interval: 1,
		},
	}
	context := &MaestroContext{WorkDir: tempDir()}
	defer cleanup(context.WorkDir)

	service := NewManagedService(config, context)
	status := service.Launch()

	if next := <-status; next != ServiceStarting {
		t.Error(next)
	}
	if next := <-status; next != ServiceHealthy {
		t.Error(next)
	}
}

func TestInitServices_HandlesDependsOn(t *testing.T) {
	context := &MaestroContext{
		ConfigFile: NewConfig([]*ServiceConfig{{
			Name: "one",
			Exec: "sleep 9000",
		}, {
			Name: "two",
			Exec: "sleep 9000",
			DependsOn: []string{"one"},
		}}),
	}

	InitServices(context)

	time.Sleep(100 * time.Millisecond)

	if services["two"].Process.Status != ServiceRunning {
		t.Error(services["two"].Process.Status)
	}
}

func TestInitServices_HandlesDependsOn_WithHealthcheck(t *testing.T) {
	context := &MaestroContext{
		ConfigFile: NewConfig([]*ServiceConfig{{
			Name: "one",
			Exec: "sleep 9000",
			Healthcheck: &HealthcheckConfig{
				Cmd:      "ls /",
				Interval: 1,
				Delay:    1,
			},
		}, {
			Name: "two",
			Exec: "sleep 9000",
			DependsOn: []string{"one"},
		}}),
	}

	InitServices(context)

	time.Sleep(100 * time.Millisecond)

	if services["one"].Status != ServiceStarting {
		t.Error(services["one"].Status)
	}

	if services["two"].Status != ServiceStopped {
		t.Error(services["two"].Status)
	}

	time.Sleep(1000 * time.Millisecond)

	if services["one"].Status != ServiceHealthy {
		t.Error(services["one"].Status)
	}

	if services["two"].Status != ServiceRunning {
		t.Error(services["two"].Status)
	}
}
