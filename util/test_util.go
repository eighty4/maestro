package util

import (
	"os"
)

func MkTmpDir() string {
	dir, _ := os.MkdirTemp(os.TempDir(), "maestro-test")
	return dir
}

func RmDir(dir string) {
	_ = os.RemoveAll(dir)
}
