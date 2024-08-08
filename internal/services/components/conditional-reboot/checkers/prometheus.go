package checkers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

const PrometheusName = "prometheus"

var (
	// try to re-use clients that use the same address
	clients = map[string]v1.API{}
	mutex   sync.Mutex
)

type PrometheusChecker struct {
	name           string
	client         v1.API
	queries        map[string]string
	address        string
	clientCertFile string
	clientKeyFile  string
	wantResponse   bool
}

type PrometheusOpts func(checker *PrometheusChecker) error

func NewPrometheusChecker(name, address string, queries map[string]string, opts ...PrometheusOpts) (*PrometheusChecker, error) {
	if len(name) == 0 {
		return nil, errors.New("no 'name' supplied")
	}

	if len(queries) == 0 {
		return nil, errors.New("no 'queries' supplied")
	}

	if len(address) == 0 {
		return nil, errors.New("empty 'address' supplied")
	}

	checker := &PrometheusChecker{
		name:    name,
		queries: queries,
		address: address,
	}

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := clients[address]; !ok {
		client, err := checker.buildClient()
		if err != nil {
			return nil, fmt.Errorf("could not build prometheus client: %w", err)
		}
		clients[address] = client
	}

	var errs error
	for _, opt := range opts {
		err := opt(checker)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	checker.client = clients[address]
	return checker, errs
}

func (c *PrometheusChecker) buildClient() (v1.API, error) {
	cl := retryablehttp.NewClient()
	cl.Logger = &ZerologAdapter{}
	cl.RetryMax = 3
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig.GetClientCertificate = c.LoadTlsClientCerts
	cl.HTTPClient.Transport = transport

	client, err := api.NewClient(api.Config{
		Address: c.address,
		Client:  cl.StandardClient(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not build prometheus client: %w", err)
	}

	return v1.NewAPI(client), nil
}

func (c *PrometheusChecker) LoadTlsClientCerts(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	if len(c.clientCertFile) == 0 || len(c.clientKeyFile) == 0 {
		return nil, errors.New("no client certificates defined")
	}

	certificate, err := tls.LoadX509KeyPair(c.clientCertFile, c.clientKeyFile)
	if err != nil {
		log.Error().Err(err).Msg("user-defined client certificates could not be loaded")
	}
	return &certificate, err
}

func (c *PrometheusChecker) Name() string {
	return fmt.Sprintf("%s - %s", PrometheusName, c.name)
}

func (c *PrometheusChecker) IsHealthy(ctx context.Context) (bool, error) {
	isHealthy := false
	for name, query := range c.queries {
		result, err := c.query(ctx, name, query)
		if err != nil {
			return false, fmt.Errorf("query '%s' returned error: %w", name, err)
		}

		isHealthy = result
		if !isHealthy {
			return false, nil
		}
	}

	return isHealthy, nil
}

func (c *PrometheusChecker) query(ctx context.Context, name, query string) (bool, error) {
	result, warnings, err := c.client.Query(ctx, query, time.Now(), v1.WithTimeout(5*time.Second))
	if err != nil {
		return false, err
	}

	if len(warnings) > 0 {
		log.Warn().Str("checker", "prometheus").Msgf("warning for query '%s': %v", name, warnings)
	}

	vec := result.(model.Vector)
	return c.evaluateResponse(len(vec)), nil
}

func (c *PrometheusChecker) evaluateResponse(responseLength int) bool {
	if c.wantResponse {
		return responseLength > 0
	}

	return responseLength == 0
}

type ZerologAdapter struct {
}

// Debug logs a debug-level message
func (z *ZerologAdapter) Debug(msg string, keysAndValues ...interface{}) {
	log.Debug().Str("checker", "prometheus").Interface("details", keysAndValues).Msg(msg)
}

// Info logs an info-level message
func (z *ZerologAdapter) Info(msg string, keysAndValues ...interface{}) {
	log.Info().Str("checker", "prometheus").Interface("details", keysAndValues).Msg(msg)
}

// Warn logs a warning-level message
func (z *ZerologAdapter) Warn(msg string, keysAndValues ...interface{}) {
	log.Warn().Str("checker", "prometheus").Interface("details", keysAndValues).Msg(msg)
}

// Error logs an error-level message
func (z *ZerologAdapter) Error(msg string, keysAndValues ...interface{}) {
	log.Error().Str("checker", "prometheus").Interface("details", keysAndValues).Msg(msg)
}
