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

func Debug() bool {
	return os.Getenv("MAESTRO_DEBUG") == "true"
}

func Duration(duration time.Duration, n int8) time.Duration {
	return time.Duration(int64(duration) * int64(n))
}

func Seconds(n int8) time.Duration {
	return Duration(time.Second, n)
}
