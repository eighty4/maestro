package util

import (
	"github.com/hashicorp/logutils"
	"log"
	"os"
)

func InitLogging() {
	minLevel := "WARN"
	if Debug() {
		minLevel = "DEBUG"
	}
	InitLoggingWithMinLevel(minLevel)
}

func InitDebugLogging() {
	InitLoggingWithMinLevel("DEBUG")
}

func InitLoggingWithMinLevel(minLevel string) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(minLevel),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
}
