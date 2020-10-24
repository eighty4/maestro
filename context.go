package main

import (
	"fmt"
	"os"
)

type MaestroContext struct {
	WorkDir string
	*ConfigFile
}

func NewMaestroContext() (*MaestroContext, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get cwd: %s", err.Error())
	}
	config, err := ReadConfig(workDir)
	if err != nil {
		return nil, fmt.Errorf("could not read config: %s", err.Error())
	}
	context := &MaestroContext{
		WorkDir: workDir,
		ConfigFile: config,
	}
	return context, nil
}
