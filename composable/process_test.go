package composable

import (
	"github.com/eighty4/maestro/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcess_StartAndStop_UpdatesStatus(t *testing.T) {
	p := NewProcess("sleep", []string{"90"}, util.Cwd())
	assert.Equal(t, NotStarted, p.Status())
	go p.Start()
	assert.Equal(t, Running, <-p.StatusC())
	assert.Equal(t, Running, p.Status())
	p.Stop()
	assert.Equal(t, Stopped, <-p.StatusC())
	assert.Equal(t, Stopped, p.Status())
}
