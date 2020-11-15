package main

import (
	"fmt"
	"os"
	"path"
)

type CliOp uint8

const (
	Main = iota
	Logs
)

func CliOpString(op CliOp) string {
	switch op {
	case Main:
		return "main"
	case Logs:
		return "logs"
	default:
		return "unknown"
	}
}

type CliCommand struct {
	Op          CliOp
	ServiceName string
}

type MaestroContext struct {
	WorkDir string
	Command *CliCommand
	*ConfigFile
}

func (mc *MaestroContext) Path(relPath string) string {
	if len(relPath) == 0 {
		return mc.WorkDir
	} else {
		return path.Join(mc.WorkDir, relPath)
	}
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
	command, err := parseCommand()
	if err != nil {
		return nil, fmt.Errorf("could not parse command: %s", err.Error())
	}
	if command.Op != Main && config == nil {
		return nil, fmt.Errorf("could not find project config to run command %s. first use command 'maestro' from this directory to create a project configuration", CliOpString(command.Op))
	}
	context := &MaestroContext{
		WorkDir:    workDir,
		Command:    command,
		ConfigFile: config,
	}
	return context, nil
}

func parseCommand() (*CliCommand, error) {
	if len(os.Args) > 2 && os.Args[1] == "logs" {
		return &CliCommand{Op: Logs, ServiceName: os.Args[2]}, nil
	} else {
		return &CliCommand{Op: Main}, nil
	}
}
