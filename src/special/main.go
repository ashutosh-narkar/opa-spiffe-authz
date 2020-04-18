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
	"time"
)

// This example assumes this workload is identified by
// the SPIFFE ID: spiffe://domain.test/special

var (
	logFlag = flag.String("log", "", "path to log to (empty=stderr)")
)

const (
	serverAddress    = "db:8082"
	serverSpiffeID   = "spiffe://domain.test/db-server"
	spiffeSocketPath = "unix:///tmp/agent.sock"
	dialTimeout      = 2 * time.Minute
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

	// Set SPIFFE_ENDPOINT_SOCKET to the workload API provider socket path (SPIRE is used in this example).
	err = os.Setenv("SPIFFE_ENDPOINT_SOCKET", spiffeSocketPath)
	if err != nil {
		log.Fatalf("Unable to set SPIFFE_ENDPOINT_SOCKET env variable: %v", err)
	}

	//Setup context
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	//Create a TLS connection.
	//The client expects the server to present an SVID with the spiffeID: 'spiffe://domain.test/db-server'

	var retry int
	var conn net.Conn
	for {
		conn, err = spiffe.DialTLS(ctx, "tcp", serverAddress, spiffe.ExpectPeer(serverSpiffeID))
		//conn, err = net.Dial("tcp", serverAddress) // THIS WORKS
		var delay time.Duration

		if err == nil {
			log.Printf("Created TLS connection after %v retries", retry)
			break
		}

		log.Printf("Unable to create TLS connection: %v", err)

		delay = time.Duration(2 * time.Minute)
		timer := time.NewTimer(delay)

		select {
		case <-timer.C:
			if err != nil {
				retry++
				log.Printf("Trying to create TLS connection. Attempt %v", retry)
			}
		}
	}

	// Send a message to the server using the TLS connection
	fmt.Fprintf(conn, "Hello server\n")

	// Read server response
	for {
		status, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil && err != io.EOF {
			log.Fatalf("Unable to read server response: %v", err)
		}
		log.Printf("Server says: %q", status)
	}
	return nil
}
