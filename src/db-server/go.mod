module db-server

go 1.14

replace github.com/opa-spiffe-demo/src/opa => ../opa

require (
	github.com/opa-spiffe-demo/src/opa v0.0.0-00010101000000-000000000000
	github.com/open-policy-agent/opa v0.19.1
	github.com/spiffe/go-spiffe/v2 v2.0.0-alpha.1
)
