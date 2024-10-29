package http_server

import (
	"context"
	"errors"

	"github.com/soerenschneider/sc-agent/internal/domain/x509"
	"github.com/soerenschneider/sc-agent/internal/services/components/acme"
)

func (s *HttpServer) CertsAcmeGetCertificates(_ context.Context, request CertsAcmeGetCertificatesRequestObject) (CertsAcmeGetCertificatesResponseObject, error) {
	if s.services.Acme == nil {
		return CertsAcmeGetCertificates501ApplicationProblemPlusJSONResponse{}, nil
	}

	cert, err := s.services.Acme.GetManagedCertificateConfigs()
	if err != nil {
		if errors.Is(err, acme.ErrCertConfigNotFound) {
			return CertsAcmeGetCertificates404ApplicationProblemPlusJSONResponse{}, nil
		}
		return CertsAcmeGetCertificates500ApplicationProblemPlusJSONResponse{}, nil
	}

	return CertsAcmeGetCertificates200JSONResponse{
		Data: convertAcmeManagedCertList(cert).Data,
	}, nil
}

func (s *HttpServer) CertsAcmeGetCertificate(_ context.Context, request CertsAcmeGetCertificateRequestObject) (CertsAcmeGetCertificateResponseObject, error) {
	if s.services.Acme == nil {
		return CertsAcmeGetCertificate501ApplicationProblemPlusJSONResponse{}, nil
	}

	cert, err := s.services.Acme.GetManagedCertificateConfig(request.Id)
	if err != nil {
		if errors.Is(err, acme.ErrCertConfigNotFound) {
			return CertsAcmeGetCertificate404ApplicationProblemPlusJSONResponse{}, nil
		}
		return CertsAcmeGetCertificate500ApplicationProblemPlusJSONResponse{}, nil
	}

	var dto AcmeManagedCertificate //nolint:gosimple
	dto = convertAcmeManagedCert(cert)

	return CertsAcmeGetCertificate200JSONResponse{
		Certificate:       dto.Certificate,
		CertificateConfig: dto.CertificateConfig,
		PostHooks:         dto.PostHooks,
		StorageConfig:     dto.StorageConfig,
	}, nil
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
