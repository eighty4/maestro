package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
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
	DependsOn   []string `yaml:"depends_on"`
}

type Config struct {
	Services       []*ServiceConfig          `yaml:"-"`
	ServicesByName map[string]*ServiceConfig `yaml:"services"`
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

	for name, service := range config.ServicesByName {
		config.Services = append(config.Services, service)
		service.Name = name
		for _, dep := range service.DependsOn {
			if _, ok := config.ServicesByName[dep]; !ok {
				return nil, fmt.Errorf("%s has declared a dep on %s that does not exist", name, dep)
			}
		}
	}

	return &config, nil
}

func configFile(dir string) string {
	if dir[len(dir)-1:] == "/" {
		return dir + ".maestro"
	} else {
		return dir + "/.maestro"
	}
}
