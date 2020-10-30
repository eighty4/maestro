package main

import (
	"os/exec"
	"testing"
)

func TestLoggerWritesCommandOutputToChannel(t *testing.T) {
	command := exec.Command("echo", "foobar")
	logger := NewProcessLogger(false)
	command.Stdout = logger
	command.Stderr = logger
	result := make(chan string, 1)
	go func() {
		logStatement := <-logger.Logs
		result <- logStatement
	}()

	go func() {
		_ = command.Run()
	}()

	if r := <-result; r != "foobar\n" {
		t.Error(r)
	}
}

func TestLoggerRetrieveLines_ReturnsEmptyWhenBeyondRange(t *testing.T) {
	command := exec.Command("echo", "foo")
	logger := NewProcessLogger(false)
	command.Stdout = logger
	command.Stderr = logger
	result := make(chan string)
	go func() {
		logStatement := <-logger.Logs
		result <- logStatement
	}()

	go func() {
		_ = command.Run()
	}()

	<-result

	retrieved := logger.RetrieveLines(1, 1)
	if len(retrieved) != 0 {
		t.Error(retrieved)
	}
}

func TestLoggerRetrieveLines_RetrieveTilEnd(t *testing.T) {
	command := exec.Command("echo", "foo\nbar")
	logger := NewProcessLogger(false)
	command.Stdout = logger
	command.Stderr = logger
	result := make(chan string)
	go func() {
		logStatement := <-logger.Logs
		result <- logStatement
	}()

	go func() {
		_ = command.Run()
	}()

	<-result

	retrieved := logger.RetrieveLines(0, 2)
	if len(retrieved) != 2 {
		t.Error(len(retrieved))
	}
	if retrieved[0] != "foo" {
		t.Error(retrieved[0])
	}
	if retrieved[1] != "bar" {
		t.Error(retrieved[1])
	}
}

func TestLoggerRetrieveLines_RetrieveSubSet(t *testing.T) {
	command := exec.Command("echo", "foo\nbar\nwoo")
	logger := NewProcessLogger(false)
	command.Stdout = logger
	command.Stderr = logger
	result := make(chan string)
	go func() {
		logStatement := <-logger.Logs
		result <- logStatement
	}()

	go func() {
		_ = command.Run()
	}()

	<-result

	retrieved := logger.RetrieveLines(0, 2)
	if len(retrieved) != 2 {
		t.Error(len(retrieved))
	}
	if retrieved[0] != "foo" {
		t.Error(retrieved[0])
	}
	if retrieved[1] != "bar" {
		t.Error(retrieved[1])
	}
}

func TestLoggerRetrieveLines_RetrieveBeyondLen(t *testing.T) {
	command := exec.Command("echo", "foo\nbar\nwoo")
	logger := NewProcessLogger(false)
	command.Stdout = logger
	command.Stderr = logger
	result := make(chan string)
	go func() {
		logStatement := <-logger.Logs
		result <- logStatement
	}()

	go func() {
		_ = command.Run()
	}()

	<-result

	retrieved := logger.RetrieveLines(0, 20)
	if len(retrieved) != 3 {
		t.Error(len(retrieved))
	}
	if retrieved[0] != "foo" {
		t.Error(retrieved[0])
	}
	if retrieved[1] != "bar" {
		t.Error(retrieved[1])
	}
	if retrieved[2] != "woo" {
		t.Error(retrieved[2])
	}
}
