package main

import (
	"fmt"
	"strings"
	"sync"
)

type Logger struct {
	Prefix string
	lines  []int
	buffer []byte
	mutex  sync.Mutex
	print  bool
	Logs   chan string `json:"-"`
}

func NewProcessLogger(print bool) *Logger {
	return &Logger{
		print: print,
		Logs:  make(chan string),
	}
}

func (l *Logger) Write(b []byte) (int, error) {
	bLen := len(b)
	if bLen == 1 {
		return bLen, nil
	}
	s := string(b)
	if l.print {
		l.writeToStdout(&s, b[bLen-1] != 10)
	}
	l.writeToBuffer(b, bLen)
	l.writeToChannelReceivers(s)
	return bLen, nil
}

func (l *Logger) writeToStdout(s *string, nl bool) {
	var lines []string
	if nl {
		lines = strings.Split(*s, "\n")
	} else {
		sansSuffixNl := *s
		sansSuffixNl = sansSuffixNl[:len(sansSuffixNl)-1]
		lines = strings.Split(sansSuffixNl, "\n")
	}
	for _, line := range lines {
		fmt.Printf("%-12.12s| %s\n", l.Prefix, line)
	}
}

func (l *Logger) writeToBuffer(b []byte, bLen int) {
	l.mutex.Lock()
	l.lines = append(l.lines, len(l.buffer))
	for i, b := range b[:bLen-1] {
		if b == 10 {
			l.lines = append(l.lines, i+1)
		}
	}
	l.buffer = append(l.buffer, b...)
	l.mutex.Unlock()
}

func (l *Logger) writeToChannelReceivers(s string) {
	select {
	case l.Logs <- s:
	default:
	}
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
