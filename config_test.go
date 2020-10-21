package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func tempDir() string {
	dir, _ := ioutil.TempDir(os.TempDir(), "maestro-test")
	return dir
}

func writeConfig(config string) string {
	dir := tempDir()
	file := configFile(dir)
	_ = ioutil.WriteFile(file, []byte(config), 0644)
	return dir
}

func cleanup(dir string) {
	_ = os.RemoveAll(dir)
}

func TestReadConfig_ReturnsWithoutConfigOrError_WhenNoConfigFileMissing(t *testing.T) {
	dir := tempDir()
	defer cleanup(dir)

	config, err := ReadConfig(dir)
	if config != nil {
		t.Error("config should be nil")
	} else if err != nil {
		t.Error("error should be nil but was: " + err.Error())
	}
}

func TestReadConfig_ReturnsConfig_WhenConfigFilePresent(t *testing.T) {
	dir := writeConfig("")
	defer cleanup(dir)

	config, _ := ReadConfig(dir)

	if config == nil {
		t.Error("config should not have been nil")
	} else if len(config.Services) != 0 {
		t.Error("config services should be empty")
	}
}

func TestReadConfig_ReturnsError_WhenConfigIsMalformedYaml(t *testing.T) {
	dir := writeConfig("	invalid	")
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if config != nil {
		t.Error("config should have been nil")
	} else if err == nil {
		t.Error("should have returned error")
	} else if !strings.HasPrefix(err.Error(), "failed to parse yaml") {
		t.Error("should be a parse yaml error but is: " + err.Error())
	}
}

func TestReadConfig_ReturnsConfig_WithExecCommand(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    exec: ls /
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if err != nil {
		t.Error(err)
	} else {
		serviceConfig, ok := config.ServicesByName["my-api"]
		if !ok {
			t.Error("service is not present")
		} else if serviceConfig.Exec != "ls /" {
			t.Error("exec value was " + serviceConfig.Exec)
		}
	}
}

func TestReadConfig_ReturnsConfig_WithGradleCommand(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    gradle:
      module: my-api-module
      task: run
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if err != nil {
		t.Error(err)
	} else {
		serviceConfig, ok := config.ServicesByName["my-api"]
		if !ok {
			t.Error("service is not present")
		} else if serviceConfig.Name != "my-api" {
			t.Errorf("expected name my-api, actual value was %s", serviceConfig.Name)
		} else if serviceConfig.Gradle == nil {
			t.Error("gradle config not present")
		} else if serviceConfig.Gradle.Module != "my-api-module" {
			t.Error("expected module my-api-module, actual value was " + serviceConfig.Gradle.Module)
		} else if serviceConfig.Gradle.Task != "run" {
			t.Error("expected task run, actual value was " + serviceConfig.Gradle.Task)
		}
	}
}

func TestReadConfig_ReturnsConfig_WithNpmScript(t *testing.T) {
	dir := writeConfig(`
services:
  my-ui:
    npm:
      script: start
      args: foo bar
      rel_dir: my/package
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if err != nil {
		t.Error(err)
	} else {
		serviceConfig, ok := config.ServicesByName["my-ui"]
		if !ok {
			t.Error("service is not present")
		} else if serviceConfig.Name != "my-ui" {
			t.Errorf("expected name my-ui, actual value was %s", serviceConfig.Name)
		} else if serviceConfig.Npm == nil {
			t.Error("npm config not present")
		} else if serviceConfig.Npm.Script != "start" {
			t.Error("expected script start, actual value was " + serviceConfig.Npm.Script)
		} else if serviceConfig.Npm.Args != "foo bar" {
			t.Error("expected args 'foo bar', actual value was " + serviceConfig.Npm.Args)
		} else if serviceConfig.Npm.RelDir != "my/package" {
			t.Error("expected rel dir my/package, actual value was " + serviceConfig.Npm.RelDir)
		}
	}
}

func TestReadConfig_ReturnsConfig_WithHealthcheck(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    exec: ls /
    healthcheck:
      cmd: ls /
      interval: 3
      delay: 3
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if err != nil {
		t.Error(err)
	} else {
		serviceConfig, ok := config.ServicesByName["my-api"]
		if !ok {
			t.Error("service is not present")
		} else if serviceConfig.Healthcheck == nil {
			t.Error("healthcheck missing")
		} else if serviceConfig.Healthcheck.Cmd != "ls /" {
			t.Error("healthcheck cmd incorrect")
		} else if serviceConfig.Healthcheck.Delay != 3 {
			t.Error("healthcheck delay incorrect")
		} else if serviceConfig.Healthcheck.Interval != 3 {
			t.Error("healthcheck interval incorrect")
		}
	}
}

func TestReadConfig_ReturnsConfig_WithDependsOnConfig(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    exec: ls /
    depends_on:
     - postgres
  postgres:
    exec: ls /
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if err != nil {
		t.Error(err)
	} else if config == nil {
		t.Error("config is nil")
	} else if len(config.Services) != 2 {
		t.Errorf("expected services count 2, actual value was %d", len(config.Services))
	} else if len(config.ServicesByName) != 2 {
		t.Errorf("expected services by name count 2, actual value was %d", len(config.ServicesByName))
	}

	serviceConfig, ok := config.ServicesByName["my-api"]
	if !ok {
		t.Error("service is not present")
	} else if serviceConfig.DependsOn[0] != "postgres" {
		t.Error("expected dep postgres, actual value was " + serviceConfig.DependsOn[0])
	}
}

func TestReadConfig_ReturnsError_WhenServiceSpecifiesMultipleExecutables(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    gradle:
      module: my-api-module
      task: run
    exec: ls /
`)
	defer cleanup(dir)

	_, err := ReadConfig(dir)

	if err == nil {
		t.Error("did not error")
	} else if err.Error() != "service my-api cannot specify multiple executable configs" {
		t.Error("err was: " + err.Error())
	}
}

func TestReadConfig_ReturnsError_WhenServiceMissingExecutable(t *testing.T) {
	dir := writeConfig(`
services:
  other-api:
    gradle:
      module: my-api-module
      task: run
  my-api:
    depends_on:
     - other-api
`)
	defer cleanup(dir)

	_, err := ReadConfig(dir)

	if err == nil {
		t.Error("did not error")
	} else if err.Error() != "service my-api missing executable config" {
		t.Error("err was: " + err.Error())
	}
}

func TestReadConfig_ReturnsError_WithHealthcheckCmdMissing(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    exec: ls /
    healthcheck:
      interval: 1
`)
	defer cleanup(dir)

	_, err := ReadConfig(dir)

	if err == nil {
		t.Error("did not error")
	} else if err.Error() != "service my-api is missing healthcheck cmd" {
		t.Error("err was: " + err.Error())
	}
}

func TestReadConfig_ReturnsError_WithHealthcheckIntervalTooLow(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    exec: ls /
    healthcheck:
      cmd: ls /
      interval: 0
`)
	defer cleanup(dir)

	_, err := ReadConfig(dir)

	if err == nil {
		t.Error("did not error")
	} else if err.Error() != "service my-api needs a healthcheck interval of 1 or greater" {
		t.Error("err was: " + err.Error())
	}
}

func TestReadConfig_ReturnsError_WithHealthcheckDelayTooLow(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    exec: ls /
    healthcheck:
      cmd: ls /
      interval: 1
      delay: -1
`)
	defer cleanup(dir)

	_, err := ReadConfig(dir)

	if err == nil {
		t.Error("did not error")
	} else if err.Error() != "service my-api needs a healthcheck delay of 1 or greater" {
		t.Error("err was: " + err.Error())
	}
}

func TestReadConfig_ReturnsError_WhenDependsOnPointsToUnknownService(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    gradle:
      module: my-api-module
      task: run
    depends_on:
     - postgres
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if config != nil {
		t.Error("config is not nil")
	} else if err == nil {
		t.Error("error is nil")
	} else if !strings.Contains(err.Error(), "has declared a dep on") {
		t.Error("error was not for missing dep")
	}
}

func TestReadConfig_ReturnsError_WhenNpmConfigIsMissingScript(t *testing.T) {
	dir := writeConfig(`
services:
  my-ui:
    npm:
      args: foo bar
      rel_dir: my/package
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if config != nil {
		t.Error("config is not nil")
	} else if err == nil {
		t.Error("error is nil")
	} else if err.Error() != "service my-ui is missing script to run" {
		t.Error("err was: " + err.Error())
	}
}

func TestReadConfig_ReturnsError_WhenGradleConfigIsMissingTask(t *testing.T) {
	dir := writeConfig(`
services:
  my-service:
    gradle:
      module: foobar
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if config != nil {
		t.Error("config is not nil")
	} else if err == nil {
		t.Error("error is nil")
	} else if err.Error() != "service my-service is missing task to run" {
		t.Error("err was: " + err.Error())
	}
}
