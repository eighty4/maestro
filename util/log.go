package util

import (
	"github.com/hashicorp/logutils"
	"log"
	"os"
	"strings"
)

func InitLogging() {
	InitLoggingWithLevel(logLevel())
}

func InitDebugLogging() {
	InitLoggingWithLevel("DEBUG")
}

func InitLoggingWithLevel(minLevel string) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"TRACE", "DEBUG", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(minLevel),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
}

func logLevel() string {
	envLogLevel := strings.ToUpper(os.Getenv("MAESTRO_LOG_LEVEL"))
	switch envLogLevel {
	case "TRACE":
		return "TRACE"
	case "DEBUG":
		return "DEBUG"
	case "WARN":
		return "WARN"
	case "ERROR":
		return "ERROR"
	default:
		if IsDebug() {
			return "DEBUG"
		} else {
			return "WARN"
		}
	}
}
