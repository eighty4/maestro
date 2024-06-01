package main

import (
	"github.com/eighty4/maestro/composable"
	"log"
	"net/http"
)

func startApiEndpoint(composition *composable.Composition) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /composable/{name}", getComposableData(composition))
	err := http.ListenAndServe("localhost:4357", mux)
	if err != nil {
		log.Fatalln("[ERROR] starting api server", err)
	} else {
		log.Println("[INFO] api listening on 4357")
	}
}

func getComposableData(composition *composable.Composition) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		composableName := request.PathValue("name")
		log.Printf("[INFO] %s\n", composableName)
		writer.Header().Add("content-type", "application/json")
		writer.WriteHeader(200)
		_, _ = writer.Write(make([]byte, 0))
	}
}
