package http_server

import (
	"cmp"
	"context"
	"errors"

	"github.com/soerenschneider/sc-agent/internal/domain/ssh"
)

func (s *HttpServer) CertsSshPostIssueRequests(ctx context.Context, request CertsSshPostIssueRequestsRequestObject) (CertsSshPostIssueRequestsResponseObject, error) {
	if s.services.SshCertificates == nil {
		return CertsSshPostIssueRequests501ApplicationProblemPlusJSONResponse{}, nil
	}

	signRequest, err := s.services.SshCertificates.GetManagedCertificateConfig(request.Params.Id)
	if err != nil {
		if errors.Is(err, ssh.ErrPkiSshConfigNotFound) {
			return CertsSshPostIssueRequests404ApplicationProblemPlusJSONResponse{}, nil
		}
		return CertsSshPostIssueRequests500ApplicationProblemPlusJSONResponse{}, nil
	}

	defaultForce := false
	forceRenewal := cmp.Or(request.Params.ForceRenewal, &defaultForce)

	signatureResult, err := s.services.SshCertificates.SignAndUpdateCert(ctx, signRequest, *forceRenewal)
	if err != nil {
		return CertsSshPostIssueRequests500ApplicationProblemPlusJSONResponse{}, nil
	}

	if signatureResult.Action == ssh.ActionNewCertificate {
		return CertsSshPostIssueRequests201Response{}, nil
	}

	return CertsSshPostIssueRequests200Response{}, nil
}

func (s *HttpServer) CertsSshGetCertificates(_ context.Context, _ CertsSshGetCertificatesRequestObject) (CertsSshGetCertificatesResponseObject, error) {
	if s.services.SshCertificates == nil {
		return CertsSshGetCertificates501ApplicationProblemPlusJSONResponse{}, nil
	}

	certs := s.services.SshCertificates.GetManagedCertificatesConfigs()

	dto := SshManagedCertificatesList{
		Data: []SshManagedCertificate{},
	}

	for _, cert := range certs {
		dto.Data = append(dto.Data, convertSshManagedCertificate(cert))
	}

	return CertsSshGetCertificates200JSONResponse{
		dto,
	}, nil
}

func (s *HttpServer) CertsSshGetCertificate(_ context.Context, request CertsSshGetCertificateRequestObject) (CertsSshGetCertificateResponseObject, error) {
	if s.services.SshCertificates == nil {
		return CertsSshGetCertificate501ApplicationProblemPlusJSONResponse{}, nil
	}

	cert, err := s.services.SshCertificates.GetManagedCertificateConfig(request.Id)
	if err != nil {
		if errors.Is(err, ssh.ErrPkiSshConfigNotFound) {
			return CertsSshGetCertificate404ApplicationProblemPlusJSONResponse{}, nil
		}

		return CertsSshGetCertificate500ApplicationProblemPlusJSONResponse{}, nil
	}

	dto := convertSshManagedCertificate(cert)
	return CertsSshGetCertificate200JSONResponse(dto), nil
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
