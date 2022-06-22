package main

type NpmScriptConfig struct {
	Script string
	Args   string
	RelDir string `yaml:"rel_dir"`
}

func (c *NpmScriptConfig) CreateProcess(context *MaestroContext) *Process {
	var process *Process
	args := []string{"run", c.Script}
	if len(c.Args) > 0 {
		args = append(args, "--", c.Args)
	}
	process = NewProcess("npm", args, context.Path(c.RelDir))
	process.Logging.print = true
	return process
}
