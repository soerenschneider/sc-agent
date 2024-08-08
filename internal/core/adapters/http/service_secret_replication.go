package http_server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/soerenschneider/sc-agent/internal/domain"
)

func (s *HttpServer) SecretsReplicationGet(w http.ResponseWriter, r *http.Request) {
	if s.services.SecretSyncer == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	configuredSyncItems := s.services.SecretSyncer.GetReplicationItems()
	marshalled, err := json.Marshal(convertSecretSyncerSyncItemsResponse(configuredSyncItems))
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(marshalled)
}

func (s *HttpServer) SecretsReplicationPost(w http.ResponseWriter, r *http.Request, params SecretsReplicationPostParams) {
	if s.services.SecretSyncer == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	syncSecretRequest, err := s.services.SecretSyncer.GetReplicationItem(params.SecretPath)
	if err != nil {
		if errors.Is(err, domain.ErrSecretReplicationItemNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "sync item not found", "")
			return
		}

		writeRfc7807Error(w, http.StatusInternalServerError, "could not retrieve sync item", "")
		return
	}

	updatedSecret, err := s.services.SecretSyncer.Replicate(r.Context(), syncSecretRequest)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "could not sync item", "")
		return
	}

	if updatedSecret {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func convertSecretSyncerSyncItemsResponse(syncItems []domain.SecretReplicationItem) SecretReplicationItems {
	ret := make([]SecretSyncerSyncItem, len(syncItems))

	for i, item := range syncItems {
		ret[i] = SecretSyncerSyncItem{
			DestUri:    item.DestUri,
			Formatter:  "",
			SecretPath: item.SecretPath,
		}
	}

	return SecretReplicationItems{Data: ret}
}
