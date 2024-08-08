package http_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	vault_x509 "github.com/soerenschneider/sc-agent/internal/services/components/vault_pki_x509"
)

func (s *HttpServer) PkiX509GetCert(w http.ResponseWriter, r *http.Request, id string) {
	if s.services.Pki == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	cert, err := s.services.Pki.GetManagedCert(id)
	if err != nil {
		if errors.Is(err, vault_x509.ErrCertConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, fmt.Sprintf("%s not found", id), "")
			return
		}
		writeRfc7807Error(w, http.StatusInternalServerError, "Internal Server Error", "")
		return
	}

	marshalled, err := json.Marshal(cert)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "Internal Server Error", "")
		return
	}

	_, _ = w.Write(marshalled)
}

func (s *HttpServer) PkiX509GetConfig(w http.ResponseWriter, r *http.Request, id string) {
	if s.services.Pki == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	conf, err := s.services.Pki.GetManagedCertConfig(id)
	if err != nil {
		if errors.Is(err, vault_x509.ErrCertConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, fmt.Sprintf("%s not found", id), "")
			return
		}
		writeRfc7807Error(w, http.StatusInternalServerError, "Internal Server Error", "")
		return
	}

	marshalled, err := json.Marshal(conf)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "Internal Server Error", "")
		return
	}

	_, _ = w.Write(marshalled)
}
