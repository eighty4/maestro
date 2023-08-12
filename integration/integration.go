package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	t.maestro = exec.Command("maestro")
	t.maestro.Dir = t.dir
	if err := t.maestro.Start(); err != nil {
		log.Fatalf("Error starting maestro process for test %s: %s\n", t.Name(), err.Error())
	}
	time.Sleep(2 * time.Second)
	t.verify = exec.Command("go", "run", "verify.go")
	t.verify.Dir = t.dir
	if err := t.verify.Run(); err != nil {
		log.Fatalf("Error running verify.go process for test %s: %s\n", t.Name(), err.Error())
	}
	if err := t.maestro.Process.Kill(); err != nil {
		log.Fatalf("Error killing maestro process for test %s: %s\n", t.Name(), err.Error())
	}
	if t.verify.ProcessState == nil {
		log.Fatalf("Error checking exit code of verify process for test %s\n", t.Name())
	}
	return t.verify.ProcessState.ExitCode() == 0
}
