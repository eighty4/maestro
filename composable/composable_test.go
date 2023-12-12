package composable

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcessStatus_String(t *testing.T) {
	assert.Equal(t, Status("NotStarted"), NotStarted)
	assert.Equal(t, Status("Starting"), Starting)
	assert.Equal(t, Status("Running"), Running)
	assert.Equal(t, Status("Stopped"), Stopped)
	assert.Equal(t, Status("Error"), Error)
}
