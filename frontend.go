package main

import (
	"github.com/eighty4/sse"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
)

const frontendUrl = "http://localhost:2999"

var streams []*sse.Connection

func StartFrontend() {
	if devRootDir := resolveDevFrontendRoot(); len(devRootDir) > 0 {
		go startDevServer(devRootDir)
	} else if rootDir := resolveFrontendRoot(); len(rootDir) > 0 {
		go serveBuiltFiles(path.Join(rootDir, "dist"))
	} else {
		log.Println("failed to resolve frontend root dir, set MAESTRO_FRONTEND_ROOT to start webapp")
	}
	go serveApiRoutes()
}

func resolveDevFrontendRoot() string {
	if devRootDir := os.Getenv("MAESTRO_FRONTEND_DEV"); len(devRootDir) > 0 {
		if !isValidFrontendRoot(devRootDir) {
			log.Fatalln("invalid MAESTRO_FRONTEND_DEV set")
		}
		return devRootDir
	}
	return ""
}

func resolveFrontendRoot() string {
	if envVar := os.Getenv("MAESTRO_FRONTEND_ROOT"); isValidFrontendRoot(envVar) {
		return envVar
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		output, err := exec.Command("which", "maestro").Output()
		if err != nil {
			log.Println(err)
			return ""
		}
		dir := string(output[:len(output)-9])
		if isValidFrontendRoot(path.Join(dir, "frontend")) {
			return dir
		}
	}
	return ""
}

func isValidFrontendRoot(frontendRootDir string) bool {
	if len(frontendRootDir) == 0 {
		return false
	}
	if _, err := os.Stat(frontendRootDir); err != nil {
		log.Println(err)
		return false
	}
	return true
}

// todo scrape Stdout of frontend process and call openFrontend() after frontend app server is up
func openFrontend() {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", frontendUrl).Start()
	case "linux":
		//err = exec.Command("xdg-open", frontendUrl).Start()
	case "windows":
		//err = exec.Command("rundll32", "url.dll,FileProtocolHandler", frontendUrl).Start()
	}
	if err != nil {
		log.Fatalln(err)
	}
}

func serveApiRoutes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", events)
	mux.HandleFunc("/logs", logs)
	mux.HandleFunc("/state", state)
	server := http.Server{
		Addr:    ":2998",
		Handler: mux,
	}
	log.Fatalln(server.ListenAndServe())
}

func serveBuiltFiles(dir string) {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(dir)))
	server := http.Server{
		Addr:    ":2999",
		Handler: mux,
	}
	log.Fatalln(server.ListenAndServe())
}

func startDevServer(dir string) {
	NewProcess("npm", []string{"run", "dev"}, dir).Start()
	frontendDevServer := exec.Command("npm", "run", "dev")
	frontendDevServer.Dir = dir
	frontendDevServer.Stdout = os.Stdout
	frontendDevServer.Stderr = os.Stderr
	err := frontendDevServer.Run()
	if err != nil {
		log.Fatalln(err)
	}
}

func events(w http.ResponseWriter, r *http.Request) {
	connection, err := sse.Upgrade(w, r)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	streams = append(streams, connection)
}

func logs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
	}
}

func state(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
	}
}
