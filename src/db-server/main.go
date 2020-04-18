package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/spiffe/go-spiffe/spiffe"

	"io"
	"log"
	"net"
	"os"
)

// This example assumes this workload is identified by
// the SPIFFE ID: spiffe://domain.test/db-server

var (
	addrFlag = flag.String("addr", ":8082", "address to bind the db server to")
	logFlag  = flag.String("log", "", "path to log to (empty=stderr)")
)

const (
	clientSpiffeID   = "spiffe://domain.test/special"
	spiffeSocketPath = "unix:///tmp/agent.sock"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	flag.Parse()
	log.SetPrefix("db> ")
	log.SetFlags(log.Ltime)
	if *logFlag != "" {
		logFile, err := os.OpenFile(*logFlag, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("unable to open log file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.SetOutput(os.Stdout)
	}

	log.Printf("starting db server...")

	// Set SPIFFE_ENDPOINT_SOCKET to the workload API provider socket path (SPIRE is used in this example).
	err = os.Setenv("SPIFFE_ENDPOINT_SOCKET", spiffeSocketPath)
	if err != nil {
		log.Fatalf("Unable to set SPIFFE_ENDPOINT_SOCKET env variable: %v", err)
	}

	// Creates a TLS listener
	// The server expects the client to present an SVID with the spiffeID: 'spiffe://domain.test/special'
	listener, err := spiffe.ListenTLS(ctx, "tcp", *addrFlag, spiffe.ExpectPeer(clientSpiffeID))
	//listener, err := net.Listen("tcp", *addrFlag) // THIS WORKS

	if err != nil {
		log.Fatalf("Unable to create TLS listener: %v", err)
	}

	defer listener.Close()

	log.Printf("listening on %s...", listener.Addr())

	// Handle connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			go handleError(err)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer conn.Close()

	for {
		log.Print("Receive command '")

		cmd, err := rw.ReadString('\n')
		switch {
		case err == io.EOF:
			log.Println("Reached EOF - close this connection.\n   ---")
			return
		case err != nil:
			log.Println("\nError reading command. Got: '"+cmd+"'\n", err)
			return
		}

		log.Printf("Client says: %q", cmd)

		// Send a response back to the client
		_, err = conn.Write([]byte("Hello client\n"))
		if err != nil {
			log.Printf("Unable to send response: %v", err)
			return
		}
	}
}

func handleError(err error) {
	log.Printf("Unable to accept connection: %v", err)
}
