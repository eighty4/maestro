package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

type HealthcheckConfig struct {
	Cmd      string
	Interval int8
	Delay    int8
}

type GradleTaskConfig struct {
	Module string
	Task   string
}

type ServiceConfig struct {
	Name        string
	Gradle      *GradleTaskConfig
	Healthcheck *HealthcheckConfig
}

type Config struct {
	Services map[string]ServiceConfig
}

func ReadConfig(dir string) (*Config, error) {
	file := configFile(dir)
	content, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to read config file: %s", err.Error())
		}
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse yaml from %s: %s", file, err.Error())
	}

	// todo validate config

	return &config, nil
}

func configFile(dir string) string {
	if dir[len(dir)-1:] == "/" {
		return dir + ".maestro"
	} else {
		return dir + "/.maestro"
	}
}
