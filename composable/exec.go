package composable

import "strings"

// ExecDescription describes an executable process.
type ExecDescription struct {
	Binary string
	Args   []string
	Dir    string
}

// DescribeExec creates an ExecDescription.
func DescribeExec(binary string, args []string, dir string) *ExecDescription {
	return &ExecDescription{
		Binary: binary,
		Args:   args,
		Dir:    dir,
	}
}

// ParseCmdString creates an ExecDescription from a shell command.
// `ls -a /` will create an ExecDescription with "ls" for ExecDescription.Binary, and []string{"-a", "/"} for ExecDescription.Args.
func ParseCmdString(cmd string, dir string) *ExecDescription {
	execSplit := strings.Fields(cmd)
	return DescribeExec(execSplit[0], execSplit[1:], dir)
}

// Process creates an instance of Process from the ExecDescription fields.
func (ed *ExecDescription) Process() *Process {
	return NewProcess(ed.Binary, ed.Args, ed.Dir)
}
