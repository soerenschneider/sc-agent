package http_server

import (
	"context"
	"errors"

	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/domain/x509"
	"github.com/soerenschneider/sc-agent/internal/services/components/pki"
)

func (s *HttpServer) CertsX509PostIssueRequests(ctx context.Context, request CertsX509PostIssueRequestsRequestObject) (CertsX509PostIssueRequestsResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *HttpServer) CertsX509GetCertificatesList(_ context.Context, _ CertsX509GetCertificatesListRequestObject) (CertsX509GetCertificatesListResponseObject, error) {
	if s.services.Pki == nil {
		return CertsX509GetCertificatesList501ApplicationProblemPlusJSONResponse{}, nil
	}

	certs := s.services.Pki.GetManagedCertificatesConfigs()
	dto := X509ManagedCertificateList{
		Data: []X509ManagedCertificate{},
	}

	for _, cert := range certs {
		dto.Data = append(dto.Data, convertX509ManagedCert(cert))
	}

	return CertsX509GetCertificatesList200JSONResponse{
		dto,
	}, nil
}

func (s *HttpServer) CertsX509GetCertificate(ctx context.Context, request CertsX509GetCertificateRequestObject) (CertsX509GetCertificateResponseObject, error) {
	if s.services.Pki == nil {
		return CertsX509GetCertificate501ApplicationProblemPlusJSONResponse{}, nil
	}

	cert, err := s.services.Pki.GetManagedCertificateConfig(request.Id)
	if err != nil {
		if errors.Is(err, pki.ErrCertConfigNotFound) {
			return CertsX509GetCertificate404ApplicationProblemPlusJSONResponse{}, nil
		}
		return CertsX509GetCertificate500ApplicationProblemPlusJSONResponse{}, nil
	}

	dto := convertX509ManagedCert(cert)
	return CertsX509GetCertificate200JSONResponse(dto), nil
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
		CaChainFile: s.CaChainFile,
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
