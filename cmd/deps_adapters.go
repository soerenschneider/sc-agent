package main

import (
	"errors"

	"github.com/soerenschneider/sc-agent/internal/config"
	http_server "github.com/soerenschneider/sc-agent/internal/core/adapters/http"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
)

func buildApiServer(conf config.Config, services *ports.Services) (*http_server.HttpServer, error) {
	if conf.Http == nil {
		return nil, errors.New("empty http config")
	}

	var opts []http_server.WebServerOpts
	if conf.Http.TlsClientAuth {
		opts = append(opts, http_server.WithTLSClientVerification(conf.Http.TlsCertFile, conf.Http.TlsKeyFile, conf.Http.TlsCaFile))
	} else if len(conf.Http.TlsCaFile) > 0 && len(conf.Http.TlsKeyFile) > 0 {
		opts = append(opts, http_server.WithTLS(conf.Http.TlsCertFile, conf.Http.TlsKeyFile))
	}

	if len(conf.Http.AllowedUserCns) > 0 || len(conf.Http.AllowedEmails) > 0 {
		opts = append(opts, http_server.WithTLSPrincipalFilter(conf.Http.AllowedUserCns, conf.Http.AllowedEmails))
	}

	return http_server.New(conf.Http.Address, services, opts...)
}
