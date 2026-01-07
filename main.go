package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/javi11/nntp-server-mock/nntpserver"
)

func main() {
	config := nntpserver.Config{
		Address:      ":1199",
		DBPath:       "",
		CleanOnClose: false,
	}

	s, err := nntpserver.NewServerWithConfig(config)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	fmt.Printf("Server listening on %s\n", s.Addr())

	// Wait for interrupt signal for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
}
