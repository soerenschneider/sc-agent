package main

import (
	"errors"

	"github.com/soerenschneider/sc-agent/internal/config"
	http_server "github.com/soerenschneider/sc-agent/internal/core/adapters/http"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

func buildApiServer(conf config.Config, services *ports.Components) (*http_server.HttpServer, error) {
	if conf.Http == nil {
		return nil, errors.New("empty http config")
	}

	if !conf.Http.Enabled {
		return nil, nil
	}

	var opts []http_server.WebServerOpts
	if conf.Http.TlsClientAuth {
		opts = append(opts, http_server.WithTLSClientVerification(conf.Http.TlsCertFile, conf.Http.TlsKeyFile, conf.Http.TlsCaFile))
	} else if len(conf.Http.TlsCertFile) > 0 && len(conf.Http.TlsKeyFile) > 0 {
		opts = append(opts, http_server.WithTLS(conf.Http.TlsCertFile, conf.Http.TlsKeyFile))
	}

	if len(conf.Http.AllowedUserCns) > 0 || len(conf.Http.AllowedEmails) > 0 {
		opts = append(opts, http_server.WithTLSPrincipalFilter(conf.Http.AllowedUserCns, conf.Http.AllowedEmails))
	}

	return http_server.New(conf.Http.Address, services, opts...)
}

func buildMetricsServer(conf config.Config) (*metrics.MetricsServer, error) {
	if conf.Metrics == nil {
		return nil, errors.New("empty metrics config")
	}

	if !conf.Metrics.Enabled {
		return nil, nil
	}

	var opts []metrics.MetricsServerOpts
	if conf.Metrics.TlsClientAuth {
		opts = append(opts, metrics.WithTLSClientVerification(conf.Metrics.TlsCertFile, conf.Metrics.TlsKeyFile, conf.Metrics.TlsCaFile))
	} else if len(conf.Metrics.TlsCertFile) > 0 && len(conf.Metrics.TlsKeyFile) > 0 {
		opts = append(opts, metrics.WithTLS(conf.Metrics.TlsCertFile, conf.Metrics.TlsKeyFile))
	}

	return metrics.New(conf.Metrics.Address, opts...)
}
