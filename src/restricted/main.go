package main

import (
	"strings"

	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/opa-spiffe-demo/src/common"
	"net/http"

	"log"
	"net"
	"os"
)

// This example assumes this workload is identified by
// the SPIFFE ID: spiffe://domain.test/restricted

var (
	addrFlag = flag.String("addr", ":8002", "address to bind the restricted server to")
	logFlag  = flag.String("log", "", "path to log to (empty=stderr)")
)

const (
	serverAddress  = "db:8082"
	clientSpiffeID = "spiffe://domain.test/restricted"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func run() (err error) {
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
	conn := common.CreateTLSDialer(serverAddress)

	// Send a message to the server using the TLS connection
	fmt.Fprintf(conn, "Hello server\n")

	msg, err := common.ReadData(conn, clientSpiffeID)
	result := common.Result{}
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
	conn := common.CreateTLSDialer(serverAddress)

	// Send a message to the server using the TLS connection
	fmt.Fprintf(conn, "/getdata\n")

	msg, err := common.ReadDataJSON(conn, clientSpiffeID)
	result := common.Result{}
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

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Expires", "0")
		h.ServeHTTP(w, r)
	})
}
