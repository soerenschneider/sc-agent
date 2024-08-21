package http_server

import (
	"errors"
)

func WithTLS(certFile, keyFile string) func(w *HttpServer) error {
	return func(w *HttpServer) error {
		if len(certFile) == 0 {
			return errors.New("empty certfile")
		}

		if len(keyFile) == 0 {
			return errors.New("empty keyfile")
		}

		w.certFile = certFile
		w.keyFile = keyFile
		return nil
	}
}

func WithTLSPrincipalFilter(users []string, emails []string) func(w *HttpServer) error {
	return func(w *HttpServer) error {
		if len(users) == 0 && len(emails) == 0 {
			return errors.New("must supply principals")
		}

		w.principalFilter = NewPrincipalFilter(users, emails)
		return nil
	}
}

func WithTLSClientVerification(certFile, keyFile, caFile string) func(w *HttpServer) error {
	return func(w *HttpServer) error {
		if len(certFile) == 0 {
			return errors.New("empty certfile")
		}

		if len(keyFile) == 0 {
			return errors.New("empty keyfile")
		}

		if len(caFile) == 0 {
			return errors.New("empty ca-file")
		}

		w.certFile = certFile
		w.keyFile = keyFile
		w.clientCa = caFile
		return nil
	}
}
