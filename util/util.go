package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ClearTermLines navigates up n number of lines and clears terminal leaving cursor on topmost cleared lined.
func ClearTermLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[A")  // up 1
		fmt.Print("\033[2K") // clear
	}
}

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

func IsFile(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.Size() > 0
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

func Subdirectories(dir string, scanDepth int) []string {
	var dirs []string
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalln(err)
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			dirs = append(dirs, filepath.Join(dir, dirEntry.Name()))
			if scanDepth > 1 {
				dirs = append(dirs, Subdirectories(filepath.Join(dir, dirEntry.Name()), scanDepth-1)...)
			}
		}
	}
	return dirs
}

// TrimRelativePathPrefix will remove the first character of a string until all relative path prefix chars have been trimmed.
func TrimRelativePathPrefix(p string) string {
	for {
		if len(p) != 0 && (p[0] == '.' || p[0] == '/' || p[0] == '\\') {
			p = p[1:]
		} else {
			return p
		}
	}
}

func WriteFile(absPath string, bytes []byte) error {
	var f *os.File
	var err error
	if f, err = os.Create(absPath); err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			log.Fatalln(err)
		}
	}(f)
	if _, err = f.Write(bytes); err != nil {
		return err
	}
	return f.Sync()
}
