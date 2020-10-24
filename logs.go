package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const SocketFileName = ".maestro.sock"

func CreateLogSocket(context *MaestroContext) {
	socketFile := context.Path(SocketFileName)
	_ = os.Remove(socketFile)
	listener, err := net.Listen("unix", socketFile)
	if err != nil {
		log.Fatalln("failed creating socket listener: " + err.Error())
	}

	go func(){
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalln("failed accepting socket connection: " + err.Error())
			}
			go func() {
				log.Println("client connected", conn.RemoteAddr().Network())
				// todo receive service name from log client
				// todo stream logs
			}()
		}
	}()
}

const (
	DialErrNoFile = "no such file or directory"
	DialErrConnRefused = "connection refused"
)

func isDialErr(err error, suffix string) bool {
	return strings.HasSuffix(err.Error(), suffix)
}

func ConnectLogSocket(context *MaestroContext) {
	socketFile := context.Path(SocketFileName)
	conn, err := net.Dial("unix", socketFile)
	if err != nil {
		if isDialErr(err, DialErrNoFile) {
			if context.ConfigFile == nil {
				fmt.Println("this is not a maestro project dir. run 'maestro' from this dir to create one or cd to another project dir.")
			} else {
				fmt.Println("you need to start 'maestro' in this project dir before connecting a separate log terminal")
			}
		} else if isDialErr(err, DialErrConnRefused) {
			fmt.Println("'maestro' is no longer running for this project dir")
			_ = os.Remove(socketFile)
		} else {
			log.Println("failed connecting to socket: " + err.Error())
		}
		os.Exit(1)
	}

	// todo send service name to main process
	// todo stream logs

	n, err := io.Copy(os.Stdout, conn)
	if err != nil {
		log.Fatalln("failed reading from socket: " + err.Error())
	}
	fmt.Printf("%d bytes copied\n", n)
}
