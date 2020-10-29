package main

import (
	"os/exec"
	"strings"
	"sync"
)

type Logger struct {
	lines  []int
	buffer []byte
	mutex  sync.Mutex
	Logs   chan string `json:"-"`
}

func NewProcessLogger(command *exec.Cmd) *Logger {
	logger := &Logger{
		Logs: make(chan string),
	}
	// todo io.Writer impl that preserves stdout vs stderr
	command.Stdout = logger
	command.Stderr = logger
	return logger
}

func (l *Logger) Write(bytes []byte) (int, error) {
	l.mutex.Lock()
	l.lines = append(l.lines, len(l.buffer))
	for i, b := range bytes[:len(bytes)-1] {
		if b == 10 {
			l.lines = append(l.lines, i+1)
		}
	}
	l.buffer = append(l.buffer, bytes...)
	l.mutex.Unlock()
	select {
	case l.Logs <- string(bytes):
	default:
	}
	return len(bytes), nil
}

func (l *Logger) RetrieveLines(f int, n int) []string {
	if len(l.lines) <= f {
		return nil
	}
	start := l.lines[f]
	tilEnd := f+n+1 > len(l.lines)
	var read []byte
	if tilEnd {
		read = l.buffer[start : len(l.buffer)-1]
	} else {
		end := l.lines[f+n] - 1
		read = l.buffer[start:end]
	}
	return strings.Split(string(read), "\n")
}
