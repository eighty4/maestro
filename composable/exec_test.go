package composable

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseCmdString_BinaryNoArgs(t *testing.T) {
	exec := ParseCmdString("ls", ".")
	assert.Equal(t, "ls", exec.Binary)
	assert.Empty(t, exec.Args)
	assert.Equal(t, ".", exec.Dir)
}

func TestParseCmdString_BinaryWithArgs(t *testing.T) {
	exec := ParseCmdString("ls -a path", ".")
	assert.Equal(t, "ls", exec.Binary)
	assert.Equal(t, []string{"-a", "path"}, exec.Args)
	assert.Equal(t, ".", exec.Dir)
}

func TestExecDescription_Process(t *testing.T) {
	process := DescribeExec("ls", []string{"-a", "path"}, ".").Process()
	assert.Equal(t, "ls", process.Binary)
	assert.Equal(t, []string{"-a", "path"}, process.Args)
	assert.Equal(t, ".", process.Dir)
}
