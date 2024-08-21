package http_server

import (
	"cmp"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain/ssh"
)

func (s *HttpServer) CertsSshPostIssueRequests(w http.ResponseWriter, r *http.Request, params CertsSshPostIssueRequestsParams) {
	if s.services.SshCertificates == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	signRequest, err := s.services.SshCertificates.GetManagedCertificateConfig(params.Id)

	if err != nil {
		if errors.Is(err, ssh.ErrPkiSshConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "not found", "")
		} else {
			writeRfc7807Error(w, http.StatusInternalServerError, "could not sign cert", "")
		}
		return
	}

	defaultForce := false
	forceRenewal := cmp.Or(params.ForceRenewal, &defaultForce)

	signatureResult, err := s.services.SshCertificates.SignAndUpdateCert(r.Context(), signRequest, *forceRenewal)
	if err != nil {
		writeRfc7807Error(w, http.StatusNotFound, "could not sign ssh public key", "")
		return
	}

	if signatureResult.Action == ssh.ActionNewCertificate {
		w.WriteHeader(http.StatusCreated)
	}
}

func (s *HttpServer) CertsSshGetCertificates(w http.ResponseWriter, r *http.Request, params CertsSshGetCertificatesParams) {
	if s.services.SshCertificates == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	certs := s.services.SshCertificates.GetManagedCertificatesConfigs()

	var dto SshManagedCertificatesList //nolint:gosimple
	dto = SshManagedCertificatesList{
		Data: []SshManagedCertificate{},
	}
	for _, cert := range certs {
		dto.Data = append(dto.Data, convertSshManagedCertificate(cert))
	}

	marshalled, err := json.Marshal(dto)
	if err != nil {
		log.Error().Err(err).Msg("could not marshal response")
		writeRfc7807Error(w, http.StatusNotFound, "unknown", "")
		return
	}

	_, _ = w.Write(marshalled)
}

func (s *HttpServer) CertsSshGetCertificate(w http.ResponseWriter, r *http.Request, id string) {
	if s.services.SshCertificates == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	cert, err := s.services.SshCertificates.GetManagedCertificateConfig(id)
	if err != nil {
		if errors.Is(err, ssh.ErrPkiSshConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "config not found", "")
			return
		}
		writeRfc7807Error(w, http.StatusNotFound, "could not fetch signature configs", "")
		return
	}

	var dto SshManagedCertificate //nolint:gosimple
	dto = convertSshManagedCertificate(cert)
	marshalled, err := json.Marshal(dto)
	if err != nil {
		log.Error().Err(err).Msg("could not marshal response")
		writeRfc7807Error(w, http.StatusNotFound, "unknown", "")
		return
	}

	_, _ = w.Write(marshalled)
}

func convertSshCertificateConfig(conf ssh.CertificateConfig) SshCertificateConfig {
	return SshCertificateConfig{
		CertType:   conf.CertType,
		Id:         conf.Id,
		Principals: conf.Principals,
		Role:       conf.Role,
		Ttl:        conf.Ttl,
	}
}

func convertSshCertificate(cert ssh.Certificate) SshCertificateData {
	return SshCertificateData{
		CriticalOptions: cert.CriticalOptions,
		Extensions:      cert.Extensions,
		Percentage:      cert.Percentage,
		Principals:      cert.Principals,
		Serial:          int64(cert.Serial), //#nosec:G115
		Type:            cert.Type,
		ValidAfter:      cert.ValidAfter,
		ValidBefore:     cert.ValidBefore,
	}
}

func convertSshManagedCertificate(cert ssh.ManagedCertificateConfig) SshManagedCertificate {
	certConfig := convertSshCertificateConfig(*cert.CertificateConfig)
	storage := convertSshCertificateStorage(*cert.StorageConfig)
	certificate := convertSshCertificate(*cert.Certificate)
	return SshManagedCertificate{
		CertificateConfig: &certConfig,
		StorageConfig:     &storage,
		Certificate:       &certificate,
	}
}

func convertSshCertificateStorage(s ssh.CertificateStorage) SshCertificateStorage {
	return SshCertificateStorage{
		CertificateFile: s.CertificateFile,
		PublicKeyFile:   s.PublicKeyFile,
	}
}
