package main

import "testing"

func TestParseExecString_WithBinaryOnly(t *testing.T) {
	binary, args := parseExecString("foo")
	if binary != "foo" {
		t.Error("binary should have been foo but was " + binary)
	}
	if len(args) != 0 {
		t.Error("args should have been [] but was", args)
	}
}

func TestParseExecString_WithArgs(t *testing.T) {
	binary, args := parseExecString("foo bar")
	if binary != "foo" {
		t.Error("binary should have been foo but was " + binary)
	}
	if len(args) != 1 || args[0] != "bar" {
		t.Error("args should have been [bar] but was", args)
	}
}

func TestCreateExecCommand(t *testing.T) {
	cmd := createExecCmd("foo bar")
	if cmd.Path != "foo" {
		t.Error("cmd should have been foo but was " + cmd.Path)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "foo" || cmd.Args[1] != "bar" {
		t.Error("cmd.Args should have been [foo bar] but was", cmd.Args)
	}
}

func TestCreateServiceCommand_ForExecutableCommand(t *testing.T) {
	config := &ServiceConfig{
		Exec: "ls /",
	}
	dir := tempDir()
	defer cleanup(dir)
	context := &MaestroContext{dir}
	command := createServiceCommand(config, context)
	if command.Path != "/bin/ls" {
		t.Error("command should have been /bin/ls but was " + command.Path)
	}
	if len(command.Args) != 2 || command.Args[0] != "ls" || command.Args[1] != "/" {
		t.Error("command.Args should have been [foo bar] but was", command.Args)
	}
}

func TestCreateServiceCommand_ForGradleTask(t *testing.T) {
	config := &ServiceConfig{
		Gradle: &GradleTaskConfig{"my-module", "my-task"},
	}
	dir := tempDir()
	defer cleanup(dir)
	context := &MaestroContext{dir}
	command := createServiceCommand(config, context)
	if command.Path != "./gradlew" {
		t.Error("command should have been ./gradlew but was " + command.Path)
	}
	if len(command.Args) != 4 || command.Args[0] != "./gradlew" || command.Args[1] != "-q" || command.Args[2] != "--console=plain" || command.Args[3] != "my-module:my-task" {
		t.Error("command.Args should have been [./gradlew -q --console=plain my-module:my-task] but was", command.Args)
	}
}

func TestCreateServiceCommand_ForNpmScript(t *testing.T) {
	config := &ServiceConfig{
		Npm: &NpmScriptConfig{"start", "--foo=bar", "my-yarn-workspace"},
	}
	dir := tempDir()
	defer cleanup(dir)
	context := &MaestroContext{dir}
	command := createServiceCommand(config, context)
	if command.Path != "/usr/local/bin/npm" { // todo fix not portable
		t.Error("command should have been /usr/local/bin/npm but was " + command.Path)
	}
	if len(command.Args) != 5 || command.Args[0] != "npm" || command.Args[1] != "run" || command.Args[2] != "start" || command.Args[3] != "--" || command.Args[4] != "--foo=bar" {
		t.Error("command.Args should have been [npm run start -- --foo=bar] but was", command.Args)
	}
	if command.Dir != dir+"/my-yarn-workspace" {
		t.Error("command.Dir should have been $TMP_DIR/my-yarn-workspace but was", command.Dir)
	}
}

func TestServiceProcess_EmitsServiceRunningAndStopped(t *testing.T) {
	config := &ServiceConfig{
		Exec: "sleep 0",
	}
	status := NewServiceProcess(config, &MaestroContext{}).Launch()
	next := <-status
	if next != Starting {
		t.Error("should have returned Starting but was " + ServiceStatusString(next))
	}
	if next = <-status; next != Running {
		t.Error("should have returned Running but was " + ServiceStatusString(next))
	}
	if next = <-status; next != Stopped {
		t.Error("should have returned Stopped but was " + ServiceStatusString(next))
	}
}

func TestServiceProcess_EmitsServiceError(t *testing.T) {
	config := &ServiceConfig{
		Exec: "sleep 0 && (exit 1)",
	}
	status := NewServiceProcess(config, &MaestroContext{}).Launch()
	next := <-status
	if next != Starting {
		t.Error("should have returned Starting but was " + ServiceStatusString(next))
	}
	if next = <-status; next != Running {
		t.Error("should have returned Running but was " + ServiceStatusString(next))
	}
	if next = <-status; next != Error {
		t.Error("should have returned Error but was " + ServiceStatusString(next))
	}
}

func TestServiceProcess_WithHealthcheck_EmitsHealthy(t *testing.T) {
	config := &ServiceConfig{
		Exec: "sleep 1",
		Healthcheck: &HealthcheckConfig{
			Cmd:      "ls /",
			Interval: 1,
		},
	}
	status := NewServiceProcess(config, &MaestroContext{}).Launch()
	next := <-status
	if next != Starting {
		t.Error("should have returned Starting but was " + ServiceStatusString(next))
	}
	if next = <-status; next != Healthy {
		t.Error("should have returned Healthy but was " + ServiceStatusString(next))
	}
}
