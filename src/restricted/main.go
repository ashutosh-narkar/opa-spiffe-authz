package main

import (
	"bufio"
	"io"
	"strings"

	"context"
	"crypto/x509"
	"encoding/json"
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
// the SPIFFE ID: spiffe://domain.test/restricted

var (
	addrFlag = flag.String("addr", ":8002", "address to bind the restricted server to")
	logFlag  = flag.String("log", "", "path to log to (empty=stderr)")
)

// Patient holds patient info
type Patient struct {
	ID           string `json:"id,omitempty"`
	Firstname    string `json:"firstname,omitempty"`
	Lastname     string `json:"lastname,omitempty"`
	SSN          string `json:"ssn,omitempty"`
	EnrolleeType string `json:"enrollee_type,omitempty"`
}

// Result holds the final response to return to the client
type Result struct {
	Client           string    `json:"client,omitempty"`
	ConnectionStatus string    `json:"connection_status,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	Patients         []Patient `json:"patients,omitempty"`
}

const (
	serverAddress = "db:8082"
	//serverSpiffeID   = "spiffe://domain.test/db-server"
	clientSpiffeID   = "spiffe://domain.test/restricted"
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
	log.SetPrefix("restricted> ")
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

	log.Printf("starting restricted server...")

	ln, err := net.Listen("tcp", *addrFlag)
	if err != nil {
		return fmt.Errorf("unable to listen: %v", err)
	}
	defer ln.Close()

	r := chi.NewRouter()
	r.Use(noCache)
	r.Get("/connect", http.HandlerFunc(handleConnect))
	r.Get("/getdata", http.HandlerFunc(handleGetData))

	log.Printf("listening on %s...", ln.Addr())
	server := &http.Server{
		Handler: r,
	}
	return server.Serve(ln)
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	conn := makeTLSConnection()

	// Send a message to the server using the TLS connection
	fmt.Fprintf(conn, "Hello server\n")

	msg, err := readDataOnConn(conn)
	result := Result{}
	result.Client = clientSpiffeID

	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		result.ConnectionStatus = "Not Created"
		result.Reason = strings.TrimSpace(err.Error())
	} else {
		log.Printf("DB Server says: %v\n", msg)
		w.WriteHeader(http.StatusOK)
		message := fmt.Sprintf("OPA allowed request: %v", strings.TrimSpace(msg))
		result.ConnectionStatus = "Created"
		result.Reason = message
	}
	json.NewEncoder(w).Encode(result)
}

func handleGetData(w http.ResponseWriter, r *http.Request) {
	conn := makeTLSConnection()

	// Send a message to the server using the TLS connection
	fmt.Fprintf(conn, "/getdata\n")

	msg, err := readDataOnConnJson(conn)
	result := Result{}
	result.Client = clientSpiffeID

	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		result.Reason = strings.TrimSpace(err.Error())
	} else {
		log.Printf("DB Server says: %v\n", msg)
		w.WriteHeader(http.StatusOK)
		result.Patients = msg
	}
	json.NewEncoder(w).Encode(result)
}

func makeTLSConnection() net.Conn {

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
	return conn
}

func readDataOnConn(conn net.Conn) (string, error) {

	// Read server response
	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && err != io.EOF && err.Error() == "remote error: tls: bad certificate" {
		log.Printf("DB Server says => OPA denied request: unexpected peer ID %v\n\n", clientSpiffeID)
		return "", fmt.Errorf("OPA denied request: unexpected peer ID %v\n\n", clientSpiffeID)
	}
	return status, nil
}

func readDataOnConnJson(conn net.Conn) ([]Patient, error) {

	// Read server response
	decoder := json.NewDecoder(conn)
	patients := []Patient{}

	if err := decoder.Decode(&patients); err != nil {
		if err.Error() == "remote error: tls: bad certificate" {
			log.Printf("DB Server says => OPA denied request: unexpected peer ID %v\n\n", clientSpiffeID)
			return nil, fmt.Errorf("OPA denied request: unexpected peer ID %v\n\n", clientSpiffeID)
		}
		log.Printf("Dencoding error: %v\n", err)
	}
	return patients, nil
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
