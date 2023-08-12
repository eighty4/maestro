package main

import (
	"log"
	"net/http"
)

func main() {
	resp, err := http.Get("http://localhost:8000")
	if err != nil {
		log.Fatalln(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("http status code was %d\n", resp.StatusCode)
	}
}
