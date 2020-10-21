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
	Exec        string
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
		err = validateServiceConfig(service, config)
		if err != nil {
			return nil, err
		}
	}

	return &config, nil
}

func validateServiceConfig(service *ServiceConfig, config Config) error {
	execSpecified := len(service.Exec) == 0
	gradleSpecified := service.Gradle == nil
	if execSpecified && gradleSpecified {
		return fmt.Errorf("service %s missing executable config", service.Name)
	}
	if !execSpecified && !gradleSpecified {
		return fmt.Errorf("service %s cannot specify an executable command and a gradle task", service.Name)
	}
	for _, dep := range service.DependsOn {
		if _, ok := config.ServicesByName[dep]; !ok {
			return fmt.Errorf("%s has declared a dep on %s that does not exist", service.Name, dep)
		}
	}
	if service.Healthcheck != nil {
		if len(service.Healthcheck.Cmd) == 0 {
			return fmt.Errorf("service %s is missing healthcheck cmd", service.Name)
		}
		if service.Healthcheck.Interval < 1 {
			return fmt.Errorf("service %s needs a healthcheck interval of 1 or greater", service.Name)
		}
		if service.Healthcheck.Delay < 0 {
			return fmt.Errorf("service %s needs a healthcheck delay of 1 or greater", service.Name)
		}
	}
	return nil
}

func configFile(dir string) string {
	if dir[len(dir)-1:] == "/" {
		return dir + ".maestro"
	} else {
		return dir + "/.maestro"
	}
}
