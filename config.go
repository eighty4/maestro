package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type ConfigFile struct {
	Services map[string]*ServiceConfig
}

type ServiceConfig struct {
	Name          string
	ProcessConfig ProcessConfig
	Healthcheck   *HealthcheckConfig `json:",omitempty"`
	DependsOn     []string           `yaml:"depends_on"`
}

type ProcessConfig interface {
	CreateProcess(context *MaestroContext) *Process
}

type HealthcheckConfig struct {
	Cmd      string
	Interval int8
	Delay    int8
}

type configFileRead struct {
	Services map[string]*serviceConfigRead
}

type serviceConfigRead struct {
	Name          string
	Exec          *ExecConfig
	Npm           *NpmScriptConfig
	Gradle        *GradleTaskConfig
	ProcessConfig ProcessConfig      `yaml:"-"`
	Healthcheck   *HealthcheckConfig `json:",omitempty"`
	DependsOn     []string           `yaml:"depends_on"`
}

func ReadConfig(dir string) (*ConfigFile, error) {
	file := configFile(dir)
	content, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to read config file: %s", err.Error())
		}
	}

	var config configFileRead
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse yaml from %s: %s", file, err.Error())
	}

	for name, service := range config.Services {
		service.Name = name
		err = validateServiceConfig(service, &config)
		if err != nil {
			return nil, err
		}
	}

	err = validateResolvableServiceDependencies(&config)
	if err != nil {
		return nil, err
	}

	return exportServiceConfigRead(&config), nil
}

func exportServiceConfigRead(cfRead *configFileRead) *ConfigFile {
	services := make(map[string]*ServiceConfig)
	for name, scRead := range cfRead.Services {
		var pc ProcessConfig
		if scRead.Gradle != nil {
			pc = scRead.Gradle
		} else if scRead.Npm != nil {
			pc = scRead.Npm
		} else if scRead.Exec != nil {
			pc = scRead.Exec
		}
		scRead.ProcessConfig = pc
		s := &ServiceConfig{
			Name:          scRead.Name,
			ProcessConfig: scRead.ProcessConfig,
			Healthcheck:   scRead.Healthcheck,
			DependsOn:     scRead.DependsOn,
		}
		services[name] = s
	}
	return &ConfigFile{
		Services: services,
	}
}

func validateServiceConfig(service *serviceConfigRead, config *configFileRead) error {
	execSpecified := service.Exec != nil
	gradleSpecified := service.Gradle != nil
	npmSpecified := service.Npm != nil
	countSpecified := 0
	for _, serviceSpecified := range []bool{execSpecified, gradleSpecified, npmSpecified} {
		if serviceSpecified {
			countSpecified++
		}
	}
	if countSpecified == 0 {
		return fmt.Errorf("service %s missing executable config", service.Name)
	}
	if countSpecified > 1 {
		return fmt.Errorf("service %s cannot specify multiple executable configs", service.Name)
	}
	for _, dep := range service.DependsOn {
		if _, ok := config.Services[dep]; !ok {
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
	if service.Npm != nil && len(service.Npm.Script) == 0 {
		return fmt.Errorf("service %s is missing script to run", service.Name)
	}
	if service.Gradle != nil && len(service.Gradle.Task) == 0 {
		return fmt.Errorf("service %s is missing task to run", service.Name)
	}
	return nil
}

func validateResolvableServiceDependencies(config *configFileRead) error {
	resolvable := false
	for _, service := range config.Services {
		resolvable = resolvable || len(service.DependsOn) == 0
		for _, thisServiceDep := range service.DependsOn {
			for _, thatServiceDep := range config.Services[thisServiceDep].DependsOn {
				if service.Name == thatServiceDep {
					return fmt.Errorf(thatServiceDep + " has a circular dependency with " + thisServiceDep)
				}
			}
		}
	}
	if !resolvable {
		return fmt.Errorf("at least one service needs to be launchable without a dependency")
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
