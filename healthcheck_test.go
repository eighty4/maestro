package main

import "testing"

func TestRunHealthcheck_Passing(t *testing.T) {
	config := &HealthcheckConfig{
		Cmd:      "ls /",
		Interval: 1,
	}
	context := &MaestroContext{WorkDir: tempDir()}
	defer cleanup(context.WorkDir)

	healthcheck := NewHealthcheck(config, context)
	go healthcheck.Start()
	if status := <-healthcheck.StatusUpdate; status != HealthcheckPassing {
		t.Error(status)
	}
}

func TestRunHealthcheck_Failing(t *testing.T) {
	config := &HealthcheckConfig{
		Cmd:      "ls humu",
		Interval: 1,
	}
	context := &MaestroContext{WorkDir: tempDir()}
	defer cleanup(context.WorkDir)

	healthcheck := NewHealthcheck(config, context)
	go healthcheck.Start()
	if status := <-healthcheck.StatusUpdate; status != HealthcheckFailing {
		t.Error(status)
	}
}
