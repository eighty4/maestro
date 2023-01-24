package util

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCwd(t *testing.T) {
	cwd, err := os.Getwd()
	assert.Nil(t, err)
	assert.Equal(t, cwd, Cwd())
}
