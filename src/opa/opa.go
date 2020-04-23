// Package opa implements helper functions to evaluate a Rego policy.
package opa

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/open-policy-agent/opa/rego"
	"io/ioutil"
	"log"
)

// policyFileName is the name of the file where the policy is defined.
const policyFileName = "policy.rego"

// Authorizer authorizes the workload given the SPIFFE ID and the chain of trust
func Authorizer(peerID string, _ [][]*x509.Certificate) error {
	input := map[string]interface{}{"peerID": peerID}
	log.Printf("OPA Input: %v", input)

	// load policy
	module, err := ioutil.ReadFile(policyFileName)
	if err != nil {
		return fmt.Errorf("failed to read policy: %v", err)
	}

	decision, err := eval(context.Background(), "data.example.allow", input, module)
	if err != nil {
		return err
	}

	switch x := decision.(type) {
	case bool:
		if x {
			log.Printf("OPA allowed request: peer ID %v", input["peerID"])
			return nil
		} else {
			return fmt.Errorf("OPA denied request: unexpected peer ID %v", input["peerID"])
		}
	default:
		return fmt.Errorf("illegal value for policy evaluation result: %T", x)
	}
}

// GetPiiFromPolicy evaluates a Rego policy and returns the PII fields
func GetPiiFromPolicy(peerID string) ([]interface{}, error) {
	input := map[string]interface{}{"peerID": peerID}
	log.Printf("OPA Input: %v", input)

	// load policy
	module, err := ioutil.ReadFile(policyFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy: %v", err)
	}

	decision, err := eval(context.Background(), "data.example.pii", input, module)
	if err != nil {
		return nil, err
	}

	switch x := decision.(type) {
	case []interface{}:
		return x, nil
	default:
		return nil, fmt.Errorf("illegal value for policy evaluation result: %T", x)
	}
}

// eval evaluates OPA query
func eval(ctx context.Context, query string, input map[string]interface{}, policy []byte) (interface{}, error) {

	// Create a new query
	r := rego.New(
		rego.Query(query),
		rego.Module(policyFileName, string(policy)),
		rego.Input(input),
	)

	// Run evaluation
	rs, err := r.Eval(ctx)

	if err != nil {
		return nil, err
	} else if len(rs) == 0 {
		return nil, fmt.Errorf("undefined decision")
	} else if len(rs) > 1 {
		return nil, fmt.Errorf("multiple evaluation results")
	}

	// Inspect results
	return rs[0].Expressions[0].Value, nil
}
