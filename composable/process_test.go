package composable

import (
	"github.com/eighty4/maestro/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcess_Start_UpdatesStatusAfterComplete(t *testing.T) {
	p := NewProcess("sleep", []string{"1"}, util.Cwd())
	defer p.Stop()
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Starting, p.Status())
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
	assert.Equal(t, Stopped, <-p.StatusC())
	assert.Equal(t, Stopped, p.Status())
}

func TestProcess_Start_UpdatesStatusAfterCancel(t *testing.T) {
	p := NewProcess("sleep", []string{"90"}, util.Cwd())
	defer p.Stop()
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Starting, p.Status())
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
	assert.Nil(t, p.Command.Cancel())
	assert.Equal(t, Stopped, <-p.StatusC())
	assert.Equal(t, Stopped, p.Status())
}

func TestProcess_Start_UpdatesStatusAfterError(t *testing.T) {
	p := NewProcess("rm", []string{"/"}, util.Cwd())
	defer p.Stop()
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Starting, p.Status())
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
	assert.Equal(t, Error, <-p.StatusC())
	assert.Equal(t, Error, p.Status())
}

func TestProcess_Stop_UpdatesStatus(t *testing.T) {
	p := NewProcess("sleep", []string{"90"}, util.Cwd())
	defer p.Stop()
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Starting, p.Status())
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
	go p.Stop()
	assert.Equal(t, Stopped, <-p.StatusC())
	assert.Equal(t, Stopped, p.Status())
}

func TestProcess_Restart_UpdatesStatus(t *testing.T) {
	p := NewProcess("sleep", []string{"90"}, util.Cwd())
	defer p.Stop()
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Starting, p.Status())
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
	go p.Restart()
	assert.Equal(t, Stopped, <-p.StatusC())
	assert.Equal(t, Stopped, p.Status())
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Starting, p.Status())
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
}

func TestProcess_Restart_Madness(t *testing.T) {
	p := NewProcess("sleep", []string{"90"}, util.Cwd())
	defer p.Stop()
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Starting, p.Status())
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
	go p.Restart()
	go p.Restart()
	go p.Restart()
	go p.Restart()
	go p.Restart()
	go p.Restart()
}

func TestProcess_Environment(t *testing.T) {
	p := NewProcess("sleep", []string{"90"}, util.Cwd())
	defer p.Stop()
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Starting, <-p.StatusC())
	assert.Equal(t, Running, <-p.StatusC())
	env := p.Environment()
	assert.NotEmpty(t, env)
	assert.Equal(t, env["COLORTERM"], "truecolor")
}
