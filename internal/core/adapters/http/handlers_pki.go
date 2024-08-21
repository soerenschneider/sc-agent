package http_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/x509"
	"github.com/soerenschneider/sc-agent/internal/services/components/pki"
)

func (s *HttpServer) CertsX509PostIssueRequests(w http.ResponseWriter, r *http.Request, params CertsX509PostIssueRequestsParams) {
	// TODO
}

func (s *HttpServer) CertsX509GetCertificatesList(w http.ResponseWriter, r *http.Request) {
	if s.services.Pki == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	certs := s.services.Pki.GetManagedCertificatesConfigs()

	var dto X509ManagedCertificateList //nolint:gosimple
	dto = X509ManagedCertificateList{
		Data: []X509ManagedCertificate{},
	}

	for _, cert := range certs {
		dto.Data = append(dto.Data, convertX509ManagedCert(cert))
	}

	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "Internal Server Error", "")
		return
	}

	_, _ = w.Write(marshalled)
}

func (s *HttpServer) CertsX509GetCertificate(w http.ResponseWriter, r *http.Request, id string) {
	if s.services.Pki == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	cert, err := s.services.Pki.GetManagedCertificateConfig(id)
	if err != nil {
		if errors.Is(err, pki.ErrCertConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, fmt.Sprintf("%s not found", id), "")
			return
		}
		writeRfc7807Error(w, http.StatusInternalServerError, "Internal Server Error", "")
		return
	}

	var dto X509ManagedCertificate //nolint:gosimple
	dto = convertX509ManagedCert(cert)
	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "Internal Server Error", "")
		return
	}

	_, _ = w.Write(marshalled)
}

func convertX509CertificateConfig(conf x509.CertificateConfig) X509CertificateConfig {
	return X509CertificateConfig{
		AltNames:   conf.AltNames,
		CommonName: conf.CommonName,
		Id:         conf.Id,
		IpSans:     conf.IpSans,
		Role:       conf.Role,
		Ttl:        conf.Ttl,
	}
}

func convertX509Certificate(cert x509.Certificate) X509CertificateData {
	return X509CertificateData{
		Issuer: PkiIssuer{
			CommonName:   cert.Issuer.CommonName,
			SerialNumber: cert.Issuer.SerialNumber,
		},
		EmailAddresses: cert.EmailAddresses,
		NotAfter:       cert.NotAfter,
		NotBefore:      cert.NotBefore,
		Serial:         cert.Serial,
		Subject:        cert.Subject,
		Percentage:     cert.Percentage,
	}
}

func convertX509Storage(s x509.CertificateStorage) X509CertificateStorage {
	return X509CertificateStorage{
		CaFile:   s.CaFile,
		CertFile: s.CertFile,
		KeyFile:  s.KeyFile,
	}
}

func convertPosthooks(hooks []domain.PostHook) []PostHooks {
	postHooks := make([]PostHooks, len(hooks))
	for idx := range hooks {
		postHooks[idx] = convertX509PostHook(hooks[idx])
	}
	return postHooks
}

func convertX509PostHook(s domain.PostHook) PostHooks {
	return PostHooks{
		Cmd:  s.Cmd,
		Name: s.Name,
	}
}

func convertX509ManagedCert(cert x509.ManagedCertificateConfig) X509ManagedCertificate {
	certConfig := convertX509CertificateConfig(*cert.CertificateConfig)
	certificateData := convertX509Certificate(*cert.Certificate)

	return X509ManagedCertificate{
		CertificateConfig: &certConfig,
		StorageConfig:     convertX509StorageItems(cert.StorageConfig),
		PostHooks:         convertPosthooks(cert.PostHooks),
		CertificateData:   &certificateData,
	}
}

func convertX509StorageItems(storageConfig []x509.CertificateStorage) []X509CertificateStorage {
	storageConf := make([]X509CertificateStorage, len(storageConfig))
	for idx := range storageConfig {
		storageConf[idx] = convertX509Storage(storageConfig[idx])
	}
	return storageConf
}
