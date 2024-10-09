package http_server

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/soerenschneider/sc-agent/internal/domain/secret_replication"
)

func (s *HttpServer) ReplicationGetSecretsItem(w http.ResponseWriter, r *http.Request, id string) {
	if s.services.SecretsReplication == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	item, err := s.services.SecretsReplication.GetReplicationItem(id)
	if err != nil {
		if errors.Is(err, secret_replication.ErrSecretsReplicationItemNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "sync item not found", "")
			return
		}

		writeRfc7807Error(w, http.StatusInternalServerError, "could not retrieve sync item", "")
		return
	}

	var dto ReplicationSecretsItem //nolint:gosimple
	dto = convertSecretReplicationItem(item)
	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(marshalled)
}

func (s *HttpServer) ReplicationPostSecretsRequests(w http.ResponseWriter, r *http.Request, params ReplicationPostSecretsRequestsParams) {
	if s.services.SecretsReplication == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	syncSecretRequest, err := s.services.SecretsReplication.GetReplicationItem(params.SecretPath)
	if err != nil {
		if errors.Is(err, secret_replication.ErrSecretsReplicationItemNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "sync item not found", "")
			return
		}

		writeRfc7807Error(w, http.StatusInternalServerError, "could not retrieve sync item", "")
		return
	}

	updatedSecret, err := s.services.SecretsReplication.Replicate(r.Context(), syncSecretRequest)
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

func (s *HttpServer) ReplicationGetSecretsItemsList(w http.ResponseWriter, r *http.Request) {
	if s.services.SecretsReplication == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	configuredSyncItems := s.services.SecretsReplication.GetReplicationItems()

	var dto ReplicationSecretsItemsList //nolint:gosimple
	dto = convertSecretReplicationItems(configuredSyncItems)
	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(marshalled)
}

func convertSecretReplicationStatus(status secret_replication.SecretReplicationStatus) ReplicationSecretsItemStatus {
	switch status {
	case secret_replication.SynchronizedStatus:
		return ReplicationSecretsItemStatusSynced
	case secret_replication.FailedStatus:
		return ReplicationSecretsItemStatusFailed
	default:
		return ReplicationSecretsItemStatusUnknown
	}
}

func convertSecretReplicationItem(item secret_replication.ReplicationItem) ReplicationSecretsItem {
	status := convertSecretReplicationStatus(item.Status)
	return ReplicationSecretsItem{
		Id:         item.ReplicationConf.Id,
		DestUri:    item.ReplicationConf.DestUri,
		Formatter:  getType(item.Formatter),
		SecretPath: item.ReplicationConf.SecretPath,
		Status:     &status,
	}
}

func convertSecretReplicationItems(syncItems []secret_replication.ReplicationItem) ReplicationSecretsItemsList {
	ret := make([]ReplicationSecretsItem, len(syncItems))

	for i, item := range syncItems {
		ret[i] = convertSecretReplicationItem(item)
	}

	return ReplicationSecretsItemsList{Data: ret}
}

func getType(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}
