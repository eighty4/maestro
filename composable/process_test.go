package composable

import (
	"github.com/eighty4/maestro/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcessStatus_String(t *testing.T) {
	assert.Equal(t, ProcessStatus("NotStarted"), ProcessNotStarted)
	assert.Equal(t, ProcessStatus("Running"), ProcessRunning)
	assert.Equal(t, ProcessStatus("Stopped"), ProcessStopped)
	assert.Equal(t, ProcessStatus("Error"), ProcessError)
}

func TestProcess_StartAndStop(t *testing.T) {
	p := NewProcess("sleep", []string{"90"}, util.Cwd())
	go p.Start()
	assert.Equal(t, ProcessRunning, <-p.ProcessStatusC)
	assert.Equal(t, ProcessRunning, p.ProcessStatus)
	p.Stop()
	assert.Equal(t, ProcessStopped, <-p.ProcessStatusC)
	assert.Equal(t, ProcessStopped, p.ProcessStatus)
}
