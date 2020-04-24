module privileged

go 1.14

replace github.com/opa-spiffe-demo/src/opa => ../opa

require (
	github.com/go-chi/chi v4.1.1+incompatible
	github.com/opa-spiffe-demo/src/opa v0.0.0-00010101000000-000000000000
	github.com/open-policy-agent/opa v0.19.1
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.0.0-alpha.1
	gopkg.in/yaml.v2 v2.2.8 // indirect
)
