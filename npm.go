package main

import "github.com/eighty4/maestro/composable"

type NpmScriptConfig struct {
	Script string
	Args   string
	RelDir string `yaml:"rel_dir"`
}

func (c *NpmScriptConfig) CreateProcess(context *MaestroContext) *composable.Process {
	var process *composable.Process
	args := []string{"run", c.Script}
	if len(c.Args) > 0 {
		args = append(args, "--", c.Args)
	}
	process = composable.NewProcess("npm", args, context.Path(c.RelDir))
	return process
}
