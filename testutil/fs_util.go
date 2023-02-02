package testutil

import (
	"os"
	"path"
	"runtime"
	"testing"
)

func MkTmpDir(t *testing.T) string {
	dir, err := os.MkdirTemp(os.TempDir(), "maestro-test")
	if err != nil {
		t.Fatal(err)
	} else if runtime.GOOS == "darwin" {
		dir = "/private" + dir
	}
	return dir
}

func RmDir(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
}

func MkDir(t *testing.T, dir string) {
	if err := os.Mkdir(dir, os.FileMode(0777)); err != nil {
		t.Fatal(err)
	}
}

func MkFile(t *testing.T, dir string, name string) {
	file, err := os.OpenFile(path.Join(dir, name), os.O_RDONLY|os.O_CREATE, os.FileMode(0777))
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}

func OpenFileForWriting(t *testing.T, dir string, name string, fn func(f *os.File)) {
	openFileWithCallback(t, dir, name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, fn)
}

func OpenFileForOverwriting(t *testing.T, dir string, name string, fn func(f *os.File)) {
	openFileWithCallback(t, dir, name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fn)
}

func openFileWithCallback(t *testing.T, dir string, name string, flag int, fn func(f *os.File)) {
	if f, err := os.OpenFile(path.Join(dir, name), flag, 0600); err != nil {
		t.Fatal(err)
	} else {
		defer func(f *os.File) {
			err := f.Close()
			if err != nil {
				t.Fatal(err)
			}
		}(f)
		fn(f)
	}
}
