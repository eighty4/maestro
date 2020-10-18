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
      module: service:session
      task: run
`)
	defer cleanup(dir)

	config, err := ReadConfig(dir)

	if err != nil {
		t.Error(err)
	} else if config == nil {
		t.Error("config should not be nil")
	} else if len(config.Services) != 1 {
		t.Error("config should have exactly one service")
	} else if _, ok := config.Services["my-api"]; !ok {
		t.Error("services should have a my-api service")
	} else if config.Services["my-api"].Gradle == nil {
		t.Error("service my-api should have a gradle task config")
	} else if config.Services["my-api"].Gradle.Module != "service:session" {
		t.Error("service my-api does not have service:session for gradle module")
	} else if config.Services["my-api"].Gradle.Task != "run" {
		t.Error("service my-api does not have run for gradle task")
	}
}
