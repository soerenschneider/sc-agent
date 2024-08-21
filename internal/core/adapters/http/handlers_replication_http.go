package http_server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/soerenschneider/sc-agent/internal/domain/http_replication"
)

func (s *HttpServer) ReplicationGetHttpItemsList(w http.ResponseWriter, r *http.Request) {
	if s.services.HttpReplication == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	items, err := s.services.HttpReplication.GetReplicationItems()
	if err != nil {
		if errors.Is(err, http_replication.ErrHttpReplicationItemNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "sync items not found", "")
			return
		}

		writeRfc7807Error(w, http.StatusInternalServerError, "could not retrieve sync item", "")
		return
	}

	var dto ReplicationHttpItemsList //nolint:gosimple
	dto = convertHttpReplicationItems(items)

	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(marshalled)
}

func (s *HttpServer) ReplicationGetHttpItem(w http.ResponseWriter, r *http.Request, id string) {
	if s.services.HttpReplication == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	item, err := s.services.HttpReplication.GetReplicationItem(id)
	if err != nil {
		if errors.Is(err, http_replication.ErrHttpReplicationItemNotFound) {
			writeRfc7807Error(w, http.StatusNotFound, "sync item not found", "")
			return
		}

		writeRfc7807Error(w, http.StatusInternalServerError, "could not retrieve sync item", "")
		return
	}

	var dto ReplicationHttpItem //nolint:gosimple
	dto = convertHttpReplicationItem(item)

	marshalled, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(marshalled)
}

func convertHttpReplicationItems(items []http_replication.ReplicationItem) ReplicationHttpItemsList {
	ret := make([]ReplicationHttpItem, len(items))

	for idx := range items {
		ret[idx] = convertHttpReplicationItem(items[idx])
	}

	return ReplicationHttpItemsList{Data: ret}
}

func convertHttpReplicationItem(item http_replication.ReplicationItem) ReplicationHttpItem {
	var expectedChecksum *string
	if len(item.ReplicationConf.Sha256Sum) > 0 {
		expectedChecksum = &item.ReplicationConf.Sha256Sum
	}
	return ReplicationHttpItem{
		Id:               item.ReplicationConf.Id,
		DestUri:          item.ReplicationConf.Destination,
		Source:           item.ReplicationConf.Source,
		ExpectedChecksum: expectedChecksum,
		PostHooks:        convertPosthooks(item.PostHooks),
		Status:           convertHttpReplicationStatus(item.Status),
	}
}

func convertHttpReplicationStatus(status http_replication.Status) ReplicationHttpItemStatus {
	switch status {
	case http_replication.InvalidChecksum:
		return ReplicationHttpItemStatusInvalidChecksum
	case http_replication.FailedStatus:
		return ReplicationHttpItemStatusFailed
	case http_replication.Synced:
		return ReplicationHttpItemStatusSynced
	default:
		return ReplicationHttpItemStatusUnknown
	}
}
