package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dirEntries, err := os.ReadDir(cwd)
	if err != nil {
		log.Fatal(err)
	}
	var tests []IntegrationTest
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			tests = append(tests, NewIntegrationTest(filepath.Join(cwd, dirEntry.Name())))
		}
	}
	m := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(tests))
	failed := 0
	fmt.Printf("%d integration tests starting\n\n", len(tests))
	for _, test := range tests {
		test := test
		go func() {
			status := "FAIL"
			if test.Run() {
				status = "PASS"
			} else {
				failed++
			}
			m.Lock()
			fmt.Printf("  %s %s\n", status, test.Name())
			m.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	if failed == 0 {
		fmt.Println("\nall tests passed")
	} else {
		fmt.Printf("\n%d tests failed", failed)
		os.Exit(1)
	}
}

type IntegrationTest struct {
	dir     string
	maestro *exec.Cmd
	verify  *exec.Cmd
}

func NewIntegrationTest(dir string) IntegrationTest {
	return IntegrationTest{
		dir: dir,
	}
}

func (t *IntegrationTest) Name() string {
	return filepath.Base(t.dir)
}

func (t *IntegrationTest) Run() bool {
	bin := "maestro"
	if runtime.GOOS == "windows" {
		bin = "maestro.exe"
	}
	t.maestro = exec.Command(bin)
	t.maestro.Env = os.Environ()
	t.maestro.Env = append(t.maestro.Env, "MAESTRO_ORCHESTRATION=1")
	t.maestro.Dir = t.dir
	maestroStdout := bytes.Buffer{}
	t.maestro.Stdout = &maestroStdout
	maestroStderr := bytes.Buffer{}
	t.maestro.Stderr = &maestroStderr
	if err := t.maestro.Start(); err != nil {
		log.Fatalf("Error starting maestro process for test %s: %s\n", t.Name(), err.Error())
	}
	// todo use healthchecks and wait for `maestro` to print the app is up and running
	time.Sleep(time.Duration(6) * time.Second)
	if t.maestro.ProcessState != nil && t.maestro.ProcessState.Exited() {
		log.Fatalf("Error running maestro with early exit for test %s with exit code: %d\n", t.Name(), t.maestro.ProcessState.ExitCode())
	}
	t.verify = exec.Command("go", "run", "verify.go")
	t.verify.Dir = t.dir
	verifyStdout := bytes.Buffer{}
	t.verify.Stdout = &verifyStdout
	verifyStderr := bytes.Buffer{}
	t.verify.Stderr = &verifyStderr
	if err := t.verify.Run(); err != nil {
		log.Fatalf("Error running verify.go process for test %s: %s\n\nmaestro STDOUT:\n%s\n\nmaestro STDERR:\n%s\n\ngo run verify.go STDOUT:\n%s\n\ngo run verify.go STDERR:\n%s\n", t.Name(), err.Error(), maestroStdout.String(), maestroStderr.String(), verifyStdout.String(), verifyStderr.String())
	}
	if err := t.maestro.Process.Kill(); err != nil {
		log.Fatalf("Error killing maestro process for test %s: %s\n", t.Name(), err.Error())
	}
	if t.verify.ProcessState == nil {
		log.Fatalf("Error checking exit code of verify process for test %s\n", t.Name())
	}
	return t.verify.ProcessState.ExitCode() == 0
}
