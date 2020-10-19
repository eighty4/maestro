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

func TestReadConfig_ReturnsConfig_WithGradleRunTask(t *testing.T) {
	dir := writeConfig(`
services:
  my-api:
    gradle:
      module: my-api-module
      task: run
    depends_on:
     - postgres
  postgres:
    gradle:
      module: postgres
      task: run
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if err != nil {
		t.Error(err)
	} else if config == nil {
		t.Error("config is nil")
	} else if len(config.Services) != 2 {
		t.Error("should have exactly two services")
	}

	serviceConfig, ok := config.Services["my-api"]
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
	} else if len(serviceConfig.DependsOn) == 0 {
		t.Error("deps is empty")
	} else if serviceConfig.DependsOn[0] != "postgres" {
		t.Error("expected dep postgres, actual value was " + serviceConfig.DependsOn[0])
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
