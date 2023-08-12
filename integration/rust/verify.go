package main

import (
	"log"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8001")
	if err != nil {
		log.Fatalf("error creating tcp client: %s\n", err.Error())
	}
	b := make([]byte, 5)
	n, err := conn.Read(b)
	if err != nil {
		log.Fatalf("error reading from tcp client: %s\n", err.Error())
	}
	if n != 5 || string(b) != "hello" {
		log.Fatalln("error receiving correct data")
	}
}
