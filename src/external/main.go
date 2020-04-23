package main

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"context"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/opa-spiffe-demo/src/opa"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"net/http"

	"log"
	"net"
	"os"
	"time"
)

// This example assumes this workload is identified by
// the SPIFFE ID: spiffe://domain.test/external

var (
	addrFlag = flag.String("addr", ":8003", "address to bind the external server to")
	logFlag  = flag.String("log", "", "path to log to (empty=stderr)")
)

const (
	serverAddress = "db:8082"
	//serverSpiffeID   = "spiffe://domain.test/db-server"
	clientSpiffeID   = "spiffe://domain.test/external"
	spiffeSocketPath = "unix:///tmp/agent.sock"
	dialTimeout      = 2 * time.Minute
)

// Result holds the final response to return to the client
type Result struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	flag.Parse()
	log.SetPrefix("external> ")
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

	log.Printf("starting external server...")

	ln, err := net.Listen("tcp", *addrFlag)
	if err != nil {
		return fmt.Errorf("unable to listen: %v", err)
	}
	defer ln.Close()

	r := chi.NewRouter()
	r.Use(noCache)
	r.Get("/connect", http.HandlerFunc(handleConnect))

	log.Printf("listening on %s...", ln.Addr())
	server := &http.Server{
		Handler: r,
	}
	return server.Serve(ln)
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	msg, err := makeTLSConnection()
	result := Result{}

	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		result.Error = strings.TrimSpace(err.Error())
	} else {
		w.WriteHeader(http.StatusOK)
		message := fmt.Sprintf("OPA allowed request: %v", strings.TrimSpace(msg))
		result.Message = message
	}
	json.NewEncoder(w).Encode(result)
}

func makeTLSConnection() (string, error) {

	// Set SPIFFE_ENDPOINT_SOCKET to the workload API provider socket path (SPIRE is used in this example).
	err := os.Setenv("SPIFFE_ENDPOINT_SOCKET", spiffeSocketPath)
	if err != nil {
		log.Fatalf("Unable to set SPIFFE_ENDPOINT_SOCKET env variable: %v", err)
	}

	//Setup context
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	//Create a TLS connection

	// allow any SPIFFE ID
	//conn, err = spiffetls.Dial(ctx, "tcp", serverAddress, tlsconfig.AuthorizeAny())

	// allow a specific SPIFFE ID
	//spiffeID, _ := spiffeid.FromString(serverSpiffeID)
	//conn, err = spiffetls.Dial(ctx, "tcp", serverAddress, tlsconfig.AuthorizeID(spiffeID))

	// OPA as authorizer
	conn, err := spiffetls.Dial(ctx, "tcp", serverAddress, Authorizer())
	if err != nil {
		log.Fatalf("Unable to create TLS connection: %v", err)
	}

	// Send a message to the server using the TLS connection
	fmt.Fprintf(conn, "Hello server\n")

	// Read server response
	for {
		status, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil && err != io.EOF && err.Error() == "remote error: tls: bad certificate" {
			msg := fmt.Sprintf("OPA denied request: unexpected peer ID %v\n\n", clientSpiffeID)
			log.Printf(msg)
			return "", fmt.Errorf(msg)
		}
		log.Printf("DB Server says: %v", status)
		return status, nil
	}
}

// Authorizer authorizes the request using OPA
func Authorizer() tlsconfig.Authorizer {
	return tlsconfig.Authorizer(func(actual spiffeid.ID, verifiedChains [][]*x509.Certificate) error {
		return opa.Authorizer(actual.String(), verifiedChains)
	})
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Expires", "0")
		h.ServeHTTP(w, r)
	})
}
