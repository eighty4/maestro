package main

import (
	"github.com/eighty4/maestro/util"
	"testing"
)

func TestNewProcess_StartsAndStops(t *testing.T) {
	dir := util.MkTmpDir()
	defer util.RmDir(dir)

	process := NewProcess("ls", []string{"/"}, dir)
	if process.Status != ProcessStopped {
		t.Error(process.Status)
	}
	go process.Start()

	if status := <-process.StatusUpdate; status != ProcessRunning {
		t.Error(status)
	}
	if status := <-process.StatusUpdate; status != ProcessStopped {
		t.Error(status)
	}
}

func TestServiceProcess_StartsAndErrors(t *testing.T) {
	dir := util.MkTmpDir()
	defer util.RmDir(dir)

	process := NewProcessFromExecString("sleep 0 && (exit 1)", dir)
	go process.Start()

	if status := <-process.StatusUpdate; status != ProcessRunning {
		t.Error(status)
	}
	if status := <-process.StatusUpdate; status != ProcessError {
		t.Error(status)
	}
}

func TestNewProcess_StopProcess(t *testing.T) {
	dir := util.MkTmpDir()
	defer util.RmDir(dir)

	process := NewProcess("sleep", []string{"9000"}, dir)
	if process.Status != ProcessStopped {
		t.Error(process.Status)
	}
	go process.Start()
	if status := <-process.StatusUpdate; status != ProcessRunning {
		t.Error(status)
	}
	go process.Stop()
	if status := <-process.StatusUpdate; status != ProcessStopped {
		t.Error(status)
	}
}

func TestParseExecString_WithBinaryOnly(t *testing.T) {
	binary, args := ParseExecString("foo")
	if binary != "foo" {
		t.Error("binary should have been foo but was " + binary)
	}
	if len(args) != 0 {
		t.Error("args should have been [] but was", args)
	}
}

func TestParseExecString_WithArgs(t *testing.T) {
	binary, args := ParseExecString("foo bar")
	if binary != "foo" {
		t.Error("binary should have been foo but was " + binary)
	}
	if len(args) != 1 || args[0] != "bar" {
		t.Error("args should have been [bar] but was", args)
	}
}
