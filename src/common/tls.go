package common

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/opa-spiffe-demo/src/opa"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const (
	spiffeSocketPath = "unix:///tmp/agent.sock"
	dialTimeout      = 2 * time.Minute
)

// CreateTLSDialer creates a mTLS connection
func CreateTLSDialer(serverAddress string) net.Conn {

	// Set SPIFFE_ENDPOINT_SOCKET to the workload API provider socket path (SPIRE is used in this example).
	err := os.Setenv("SPIFFE_ENDPOINT_SOCKET", spiffeSocketPath)
	if err != nil {
		log.Fatalf("Unable to set SPIFFE_ENDPOINT_SOCKET env variable: %v", err)
	}

	//Setup context
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	// Create a TLS connection with OPA as authorizer
	conn, err := spiffetls.Dial(ctx, "tcp", serverAddress, Authorizer())
	if err != nil {
		log.Fatalf("Unable to create TLS connection: %v", err)
	}
	return conn
}

func CreateTLSLIstener(ctx context.Context, serverAddress string) net.Listener {

	// Set SPIFFE_ENDPOINT_SOCKET to the workload API provider socket path (SPIRE is used in this example).
	err := os.Setenv("SPIFFE_ENDPOINT_SOCKET", spiffeSocketPath)
	if err != nil {
		log.Fatalf("Unable to set SPIFFE_ENDPOINT_SOCKET env variable: %v", err)
	}

	// Creates a TLS listener with OPA as authorizer
	listener, err := spiffetls.Listen(ctx, "tcp", serverAddress, Authorizer())
	if err != nil {
		log.Fatalf("Unable to create TLS listener: %v", err)
	}
	return listener
}

// Authorizer authorizes the request using OPA
func Authorizer() tlsconfig.Authorizer {
	return tlsconfig.Authorizer(func(actual spiffeid.ID, verifiedChains [][]*x509.Certificate) error {
		return opa.Authorizer(actual.String(), verifiedChains)
	})
}

// ReadData reads server response
func ReadData(conn net.Conn, clientSpiffeID string) (string, error) {

	// Read server response
	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && err != io.EOF && err.Error() == "remote error: tls: bad certificate" {
		log.Printf("DB Server says => OPA denied request: unexpected peer ID %v\n\n", clientSpiffeID)
		return "", fmt.Errorf("OPA denied request: unexpected peer ID %v\n\n", clientSpiffeID)
	}
	return status, nil
}

// ReadDataJSON reads server response
func ReadDataJSON(conn net.Conn, clientSpiffeID string) ([]Patient, error) {

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
