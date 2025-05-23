package main

import (
	"fmt"
	"log"
	"net"

	"github.com/javi11/nntp-server-mock/nntpserver"
)

const (
	port = ":1199" // Usenet default port is 119, but we use 1199 for testing
)

func main() {
	a, err := net.ResolveTCPAddr("tcp", port)
	maybefatal(err, "Error resolving listener: %v", err)
	l, err := net.ListenTCP("tcp", a)
	maybefatal(err, "Error setting up listener: %v", err)
	defer l.Close()

	backend := nntpserver.NewDiskBackend(
		false,
		"",
	)
	s := nntpserver.NewServer(backend)

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
