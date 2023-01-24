package main

import (
	"github.com/eighty4/maestro/util"
	"testing"
)

func TestMain(m *testing.M) {
	util.InitDebugLogging()
	m.Run()
}
