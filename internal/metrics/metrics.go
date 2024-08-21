package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

const (
	namespace                        = "sc_agent"
	defaultMetricsDumpFrequency      = 1 * time.Minute
	defaultMetricsHeartbeatFrequency = 1 * time.Minute
	metricsServerComponent           = "metrics"
)

type MetricsServer struct {
	address string

	// optional
	certFile string
	keyFile  string
	clientCa string
}

type MetricsServerOpts func(*MetricsServer) error

func New(address string, opts ...MetricsServerOpts) (*MetricsServer, error) {
	if len(address) == 0 {
		return nil, errors.New("empty address provided")
	}

	w := &MetricsServer{
		address: address,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(w); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return w, errs
}

func (s *MetricsServer) StartServer(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		Addr:              s.address,
		Handler:           mux,
		ReadTimeout:       1 * time.Second,
		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	if s.IsTLSClientAuthConfigured() {
		var caCertPool *x509.CertPool
		caCert, err := os.ReadFile(s.clientCa)
		if err != nil {
			return err
		}
		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		server.TLSConfig = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  caCertPool,
			MinVersion: tls.VersionTLS13,
		}
	}

	errChan := make(chan error)
	go func() {
		log.Info().Str("component", metricsServerComponent).Str("address", s.address).Msg("Starting server")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("can not start metrics server: %w", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("component", metricsServerComponent).Msg("Stopping server")
			return server.Shutdown(ctx)
		case err := <-errChan:
			return err
		}
	}
}

func (s *MetricsServer) IsTLSConfigured() bool {
	return len(s.certFile) > 0 && len(s.keyFile) > 0
}

func (s *MetricsServer) IsTLSClientAuthConfigured() bool {
	return s.IsTLSConfigured() && len(s.clientCa) > 0
}
