package main

import (
	"github.com/eighty4/maestro/util"
	"testing"
	"time"
)

func newTestConfig(services []*ServiceConfig) *ConfigFile {
	servicesByName := make(map[string]*ServiceConfig)
	for _, service := range services {
		servicesByName[service.Name] = service
	}
	return &ConfigFile{
		Services: servicesByName,
	}
}

func TestCreateServiceCommand_ForExecutableCommand(t *testing.T) {
	config := &ServiceConfig{
		ProcessConfig: &ExecConfig{
			Cmd: "ls /",
		},
	}
	context := &MaestroContext{WorkDir: util.MkTmpDir()}
	defer util.RmDir(context.WorkDir)

	command := config.ProcessConfig.CreateProcess(context)
	if command.Binary != "ls" {
		t.Error(command.Binary)
	}
	if len(command.Args) != 1 || command.Args[0] != "/" {
		t.Error(command.Args)
	}
}

func TestCreateServiceCommand_ForGradleTask(t *testing.T) {
	config := &ServiceConfig{
		ProcessConfig: &GradleTaskConfig{"my-module", "my-task"},
	}
	context := &MaestroContext{WorkDir: util.MkTmpDir()}
	defer util.RmDir(context.WorkDir)

	command := config.ProcessConfig.CreateProcess(context)
	if command.Binary != "./gradlew" {
		t.Error(command.Binary)
	}
	if len(command.Args) != 3 || command.Args[0] != "-q" || command.Args[1] != "--console=plain" || command.Args[2] != "my-module:my-task" {
		t.Error(command.Args)
	}
}

func TestCreateServiceCommand_ForNpmScript(t *testing.T) {
	config := &ServiceConfig{
		ProcessConfig: &NpmScriptConfig{"start", "--foo=bar", "my-yarn-workspace"},
	}
	context := &MaestroContext{WorkDir: util.MkTmpDir()}
	defer util.RmDir(context.WorkDir)

	command := config.ProcessConfig.CreateProcess(context)
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
		ProcessConfig: &ExecConfig{
			Cmd: "sleep 2",
		},
	}
	context := &MaestroContext{WorkDir: util.MkTmpDir()}
	defer util.RmDir(context.WorkDir)

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
		ProcessConfig: &ExecConfig{
			Cmd: "sleep 2",
		},
		Healthcheck: &HealthcheckConfig{
			Cmd:      "ls /",
			Interval: 1,
		},
	}
	context := &MaestroContext{WorkDir: util.MkTmpDir()}
	defer util.RmDir(context.WorkDir)

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
	t.Skip("overhaul 'refactor' to fix")
	context := &MaestroContext{
		ConfigFile: newTestConfig([]*ServiceConfig{{
			Name: "one",
			ProcessConfig: &ExecConfig{
				Cmd: "sleep 9000",
			},
		}, {
			Name: "two",
			ProcessConfig: &ExecConfig{
				Cmd: "sleep 9000",
			},
			DependsOn: []string{"one"},
		}}),
	}

	orchestration := NewServiceOrchestration(context)

	time.Sleep(100 * time.Millisecond)

	if orchestration.Services["two"].Process.Status != ServiceRunning {
		t.Error(orchestration.Services["two"].Process.Status)
	}
}

func TestInitServices_HandlesDependsOn_WithHealthcheck(t *testing.T) {
	t.Skip("overhaul 'refactor' to fix")
	context := &MaestroContext{
		ConfigFile: newTestConfig([]*ServiceConfig{{
			Name: "one",
			ProcessConfig: &ExecConfig{
				Cmd: "sleep 9000",
			},
			Healthcheck: &HealthcheckConfig{
				Cmd:      "ls /",
				Interval: 1,
				Delay:    1,
			},
		}, {
			Name: "two",
			ProcessConfig: &ExecConfig{
				Cmd: "sleep 9000",
			},
			DependsOn: []string{"one"},
		}}),
	}

	orchestration := NewServiceOrchestration(context)

	time.Sleep(100 * time.Millisecond)

	if orchestration.Services["one"].Status != ServiceStarting {
		t.Error(orchestration.Services["one"].Status)
	}

	if orchestration.Services["two"].Status != ServiceStopped {
		t.Error(orchestration.Services["two"].Status)
	}

	time.Sleep(1000 * time.Millisecond)

	if orchestration.Services["one"].Status != ServiceHealthy {
		t.Error(orchestration.Services["one"].Status)
	}

	if orchestration.Services["two"].Status != ServiceRunning {
		t.Error(orchestration.Services["two"].Status)
	}
}
