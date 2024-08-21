package checkers

import (
	"errors"
)

func ExceptsResponse(expectsResponse bool) PrometheusOpts {
	return func(checker *PrometheusChecker) error {
		checker.wantResponse = expectsResponse
		return nil
	}
}

func UseTls(certFile, keyFile string) PrometheusOpts {
	return func(checker *PrometheusChecker) error {
		checker.clientCertFile = certFile
		checker.clientKeyFile = keyFile
		return nil
	}
}

//nolint:cyclop
func PrometheusCheckerFromMap(args map[string]any) (*PrometheusChecker, error) {
	if len(args) == 0 {
		return nil, errors.New("could not build prometheus checker, empty args supplied")
	}

	name, ok := args["name"].(string)
	if !ok {
		return nil, errors.New("could not build prometheus checker, empty 'name' provided")
	}

	address, ok := args["address"].(string)
	if !ok {
		return nil, errors.New("could not build prometheus checker, empty 'address' provided")
	}

	queries, ok := args["queries"]
	if !ok {
		return nil, errors.New("could not build prometheus checker, empty 'queries' provided")
	}
	queriesTmp, ok := queries.(map[string]any)
	if !ok {
		return nil, errors.New("'queries' is not of type map[string]string")
	}
	queriesMap := map[string]string{}
	for k := range queriesTmp {
		if v, ok := queriesTmp[k].(string); ok {
			queriesMap[k] = v
		}
	}

	var opts []PrometheusOpts
	wantResponse, ok := args["wantResponse"].(bool)
	if !ok {
		opts = append(opts, ExceptsResponse(wantResponse))
	}

	clientCert, okCert := args["tls_client_cert"].(string)
	clientKey, okKey := args["tls_client_key"].(string)
	if okCert && okKey {
		opts = append(opts, UseTls(clientCert, clientKey))
	}

	return NewPrometheusChecker(name, address, queriesMap, opts...)
}
