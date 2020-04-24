package main

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/opa-spiffe-demo/src/opa"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"strings"

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
	//clientSpiffeID   = "spiffe://domain.test/privileged"
	spiffeSocketPath = "unix:///tmp/agent.sock"
)

// Patient holds patient info
type Patient struct {
	ID           string `json:"id,omitempty"`
	Firstname    string `json:"firstname,omitempty"`
	Lastname     string `json:"lastname,omitempty"`
	SSN          string `json:"ssn,omitempty"`
	EnrolleeType string `json:"enrollee_type,omitempty"`
}

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

	// allow any SPIFFE ID
	//listener, err := spiffetls.Listen(ctx, "tcp", *addrFlag, tlsconfig.AuthorizeAny())

	// allow a specific SPIFFE ID
	//spiffeID, _ := spiffeid.FromString(clientSpiffeID)
	//listener, err := spiffetls.Listen(ctx, "tcp", *addrFlag, tlsconfig.AuthorizeID(spiffeID))

	// OPA as authorizer
	listener, err := spiffetls.Listen(ctx, "tcp", *addrFlag, Authorizer())

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
		cmd, err := rw.ReadString('\n')
		switch {
		case err == io.EOF:
			log.Println("Reached EOF - close this connection.\n   ---")
			return
		case err != nil:
			log.Printf("Error: %v\n", err)
			return
		}

		log.Printf("Client says: %q", cmd)

		// Send a response back to the client
		if strings.HasPrefix(cmd, "/getdata") {
			data := generateTestData()
			data = getObfuscateResult(conn, data)

			encoder := json.NewEncoder(conn)

			if err := encoder.Encode(data); err != nil {
				log.Printf("Encoding error: %v\n", err)
			}
		} else {
			id, _ := spiffetls.PeerIDFromConn(conn)
			_, err = conn.Write([]byte(fmt.Sprintf("Hello %v\n", id)))
			if err != nil {
				log.Printf("Unable to send response: %v", err)
				return
			}
		}
	}
}

func handleError(err error) {
	log.Printf("Unable to accept connection: %v", err)
}

func generateTestData() []Patient {
	patients := []Patient{}
	patients = append(patients, Patient{
		ID:           "1",
		Firstname:    "Iron",
		Lastname:     "Man",
		SSN:          "111-11-1111",
		EnrolleeType: "Primary",
	})

	patients = append(patients, Patient{
		ID:           "2",
		Firstname:    "Thor",
		Lastname:     "Odinson",
		SSN:          "222-22-2222",
		EnrolleeType: "Primary",
	})

	patients = append(patients, Patient{
		ID:           "3",
		Firstname:    "Peter",
		Lastname:     "Parker",
		SSN:          "333-33-3333",
		EnrolleeType: "Secondary",
	})

	patients = append(patients, Patient{
		ID:           "4",
		Firstname:    "Nick",
		Lastname:     "Fury",
		SSN:          "333-33-3333",
		EnrolleeType: "Secondary",
	})
	return patients
}

// Authorizer authorizes the request using OPA
func Authorizer() tlsconfig.Authorizer {
	return tlsconfig.Authorizer(func(actual spiffeid.ID, verifiedChains [][]*x509.Certificate) error {
		return opa.Authorizer(actual.String(), verifiedChains)
	})
}

func getObfuscateResult(conn net.Conn, original []Patient) []Patient {
	id, _ := spiffetls.PeerIDFromConn(conn)
	fields, err := opa.GetPiiFromPolicy(id.String())

	if err != nil {
		return []Patient{}
	}

	if len(fields) == 0 {
		return original
	}

	// filter the fields
	filterMap := make(map[string]bool)

	for _, field := range fields {
		filterMap[field.(string)] = true
	}

	// build a new result based on the fields to filter
	patients := []Patient{}

	for _, p := range original {
		newPatient := Patient{}

		if _, ok := filterMap["ID"]; ok {
			newPatient.ID = "***********"
		} else {
			newPatient.ID = p.ID
		}

		if _, ok := filterMap["Firstname"]; ok {
			newPatient.Firstname = "***********"
		} else {
			newPatient.Firstname = p.Firstname
		}

		if _, ok := filterMap["Lastname"]; ok {
			newPatient.Lastname = "***********"
		} else {
			newPatient.Lastname = p.Lastname
		}

		if _, ok := filterMap["SSN"]; ok {
			newPatient.SSN = "***********"
		} else {
			newPatient.SSN = p.SSN
		}

		if _, ok := filterMap["EnrolleeType"]; ok {
			newPatient.EnrolleeType = "***********"
		} else {
			newPatient.EnrolleeType = p.EnrolleeType
		}

		patients = append(patients, newPatient)
	}
	return patients
}
