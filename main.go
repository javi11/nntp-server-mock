package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

const (
	port        = ":1199" // Usenet default port is 119, but we use 1199 for testing
	articlesDir = "./articles"
)

func main() {
	if err := os.MkdirAll(articlesDir, 0755); err != nil {
		fmt.Printf("Error creating articles directory: %v\n", err)
		return
	}

	a, err := net.ResolveTCPAddr("tcp", port)
	maybefatal(err, "Error resolving listener: %v", err)
	l, err := net.ListenTCP("tcp", a)
	maybefatal(err, "Error setting up listener: %v", err)
	defer l.Close()

	backend := NewDiskBackend()
	s := NewServer(backend)

	fmt.Printf("Server listening on port %s\n", port)

	for {
		c, err := l.AcceptTCP()
		maybefatal(err, "Error accepting connection: %v", err)
		go s.Process(c)
	}

}

func maybefatal(err error, f string, a ...interface{}) {
	if err != nil {
		log.Fatalf(f, a...)
	}
}
