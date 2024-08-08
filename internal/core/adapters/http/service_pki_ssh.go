package http_server

import (
	"cmp"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

func (s *HttpServer) PkiSshSignaturesGet(w http.ResponseWriter, r *http.Request, id string, params PkiSshSignaturesGetParams) {
	if s.services.SshSigner == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var signRequest domain.SshSignatureRequest
	var err error
	if params.Type == PkiSshSignaturesGetParamsTypeHost {
		signRequest, err = s.services.SshSigner.GetHostRequest(id)
	} else if params.Type == PkiSshSignaturesGetParamsTypeUser {
		signRequest, err = s.services.SshSigner.GetUserRequest(id)
	}

	if err != nil {
		if errors.Is(err, domain.ErrPkiSshConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "config not found", "")
			return
		}
		writeRfc7807Error(w, http.StatusNotFound, "could not fetch signature configs", "")
		return
	}

	marshalled, err := json.Marshal(signRequest)
	if err != nil {
		log.Error().Err(err).Msg("could not marshal response")
		writeRfc7807Error(w, http.StatusNotFound, "unknown", "")
		return
	}

	_, _ = w.Write(marshalled)
}

func (s *HttpServer) PkiSshSignaturesPost(w http.ResponseWriter, r *http.Request, id string, params PkiSshSignaturesPostParams) {
	if s.services.SshSigner == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var signRequest domain.SshSignatureRequest
	var err error
	if params.Type == PkiSshSignaturesPostParamsTypeHost {
		signRequest, err = s.services.SshSigner.GetHostRequest(id)
	} else if params.Type == PkiSshSignaturesPostParamsTypeUser {
		signRequest, err = s.services.SshSigner.GetUserRequest(id)
	}

	if err != nil {
		if errors.Is(err, domain.ErrPkiSshConfigNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "no such ", "")
		} else {
			writeRfc7807Error(w, http.StatusInternalServerError, "could not sign cert", "")
		}
		return
	}

	defaultForce := false
	forceRenewal := cmp.Or(params.ForceRenewal, &defaultForce)

	signatureResult, err := s.services.SshSigner.SignAndUpdateCert(r.Context(), signRequest, *forceRenewal)
	if err != nil {
		writeRfc7807Error(w, http.StatusNotFound, "could not sign ssh public key", "")
		return
	}

	marshalled, err := json.Marshal(convertSshSignerResponse(*signatureResult))
	if err != nil {
		log.Error().Err(err).Msg("could not marshal response")
		writeRfc7807Error(w, http.StatusNotFound, "unknown", "")
		return
	}

	if signatureResult.Action == domain.ActionNewCertificate {
		w.WriteHeader(http.StatusCreated)
	}

	_, _ = w.Write(marshalled)
}

func convertSshSignerResponse(result domain.SshSignatureResult) PkiSshConfig {
	serial := int64(result.CertData.Serial)
	pct := result.CertData.GetPercentage()

	return PkiSshConfig{
		CertInfo: &SshSignerCertificateInfo{
			PublicKeyFile:   &result.PublicKeyFile,
			CertificateFile: &result.CertificateFile,
			Serial:          &serial,
			ValidAfter:      &result.CertData.ValidAfter,
			ValidBefore:     &result.CertData.ValidBefore,
			Percentage:      &pct,
		},
	}
}
