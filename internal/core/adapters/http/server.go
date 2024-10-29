package http_server

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

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"gitlab.com/tanna.dev/openapi-doc-http-handler/elements"
	"go.uber.org/multierr"
)

const (
	apiServerComponent = "api-server"
)

var _ StrictServerInterface = (*HttpServer)(nil)

type HttpServer struct {
	address string

	services *ports.Components

	// optional
	certFile string
	keyFile  string
	clientCa string

	principalFilter *TlsClientPrincipalFilter
}

func (s *HttpServer) CertsX509PostIssueRequests(ctx context.Context, request CertsX509PostIssueRequestsRequestObject) (CertsX509PostIssueRequestsResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

type WebServerOpts func(*HttpServer) error

func New(address string, services *ports.Components, opts ...WebServerOpts) (*HttpServer, error) {
	if len(address) == 0 {
		return nil, errors.New("empty address provided")
	}

	w := &HttpServer{
		address:  address,
		services: services,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(w); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return w, errs
}

func (s *HttpServer) IsTLSConfigured() bool {
	return len(s.certFile) > 0 && len(s.keyFile) > 0
}

func (s *HttpServer) IsTLSClientAuthConfigured() bool {
	return s.IsTLSConfigured() && len(s.clientCa) > 0
}

func (s *HttpServer) getOpenApiHandler() (http.Handler, error) {
	// add a mux that serves /docs
	swagger, err := GetSwagger()
	if err != nil {
		return nil, err
	}

	docs, err := elements.NewHandler(swagger, err)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/docs", docs)
	mux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	})

	options := StdHTTPServerOptions{
		Middlewares: []MiddlewareFunc{
			logStatusCodeMiddleware,
		},
		BaseRouter: mux,
	}

	if s.principalFilter != nil {
		options.Middlewares = append(options.Middlewares, s.principalFilter.tlsClientCertMiddleware)
	}

	fickdich := NewStrictHandler(s, nil)
	return HandlerWithOptions(fickdich, options), nil
}

func (s *HttpServer) StartServer(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	handler, err := s.getOpenApiHandler()
	if err != nil {
		return err
	}

	server := http.Server{
		Addr:              s.address,
		Handler:           handler,
		ReadTimeout:       3 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
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
		log.Info().Str("component", apiServerComponent).Bool("tls", s.IsTLSConfigured()).Str("address", s.address).Msg("Starting server")
		if s.IsTLSConfigured() {
			if server.TLSConfig == nil {
				server.TLSConfig = &tls.Config{
					MinVersion: tls.VersionTLS13,
				}
			}

			getCert := func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
				if err != nil {
					metrics.AdapterHttpTlsErrors.Inc()
					return nil, err
				}
				return &cert, nil
			}

			server.TLSConfig.GetCertificate = getCert
			if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- fmt.Errorf("can not start api server: %w", err)
			}
		} else {
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- fmt.Errorf("can not start api server: %w", err)
			}
		}
	}()

	select {
	case <-ctx.Done():
		log.Info().Str("component", apiServerComponent).Msg("Stopping server")
		return server.Shutdown(ctx)
	case err := <-errChan:
		return err
	}
}
