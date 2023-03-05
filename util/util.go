package util

import (
	"log"
	"os"
	"time"
)

func Cwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("os.Getwd() err", err.Error())
	}
	return cwd
}

func IsDebug() bool {
	return os.Getenv("MAESTRO_DEBUG") == "true"
}

func Duration(duration time.Duration, n int8) time.Duration {
	return time.Duration(int64(duration) * int64(n))
}

func IsDir(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}

func Seconds(n int8) time.Duration {
	return Duration(time.Second, n)
}

func PluralPrint(s string, n int) string {
	if n == 1 {
		return s
	} else {
		return s + "s"
	}
}

func SinglePrintIes(s string, n int) string {
	if n == 1 {
		return s[:len(s)-3] + "y"
	} else {
		return s
	}
}
