package http_server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/soerenschneider/sc-agent/internal/domain/x509"
	"github.com/soerenschneider/sc-agent/internal/services/components/acme"
)

func (s *HttpServer) AcmeGetManagedCerts(w http.ResponseWriter, r *http.Request) {
	if s.services.Acme == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	cert, err := s.services.Acme.GetManagedCertificateConfigs()
	if err != nil {
		if errors.Is(err, acme.ErrCertConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "not found", "")
			return
		}
		writeRfc7807Error(w, http.StatusInternalServerError, "could not stop k0s", "")
		return
	}

	var dto AcmeManagedCertificateList //nolint:gosimple
	dto = convertAcmeManagedCertList(cert)

	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(marshalled)

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "could not stop k0s", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HttpServer) AcmeGetManagedCert(w http.ResponseWriter, r *http.Request, id string) {
	if s.services.Acme == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	cert, err := s.services.Acme.GetManagedCertificateConfig(id)
	if err != nil {
		if errors.Is(err, acme.ErrCertConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "not found", "")
			return
		}
		writeRfc7807Error(w, http.StatusInternalServerError, "could not stop k0s", "")
		return
	}

	var dto AcmeManagedCertificate //nolint:gosimple
	dto = convertAcmeManagedCert(cert)

	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(marshalled)
}

func convertAcmeManagedCertList(certs []x509.ManagedCertificateConfig) AcmeManagedCertificateList {
	ret := make([]AcmeManagedCertificate, len(certs))

	for idx, cert := range certs {
		ret[idx] = convertAcmeManagedCert(cert)
	}

	return AcmeManagedCertificateList{
		Data: ret,
	}
}

func convertAcmeManagedCert(cert x509.ManagedCertificateConfig) AcmeManagedCertificate {
	certificate := convertX509Certificate(*cert.Certificate)
	return AcmeManagedCertificate{
		Certificate:       &certificate,
		CertificateConfig: nil,
		PostHooks:         convertPosthooks(cert.PostHooks),
		StorageConfig:     convertX509StorageItems(cert.StorageConfig),
	}
}
