package http_server

import (
	"context"
	"errors"

	"github.com/soerenschneider/sc-agent/internal/domain/http_replication"
)

func (s *HttpServer) ReplicationGetHttpItemsList(ctx context.Context, request ReplicationGetHttpItemsListRequestObject) (ReplicationGetHttpItemsListResponseObject, error) {
	if s.services.HttpReplication == nil {
		return ReplicationGetHttpItemsList501ApplicationProblemPlusJSONResponse{}, nil
	}

	items, err := s.services.HttpReplication.GetReplicationItems()
	if err != nil {
		if errors.Is(err, http_replication.ErrHttpReplicationItemNotFound) {
			return ReplicationGetHttpItemsList404ApplicationProblemPlusJSONResponse{}, nil
		}
		return ReplicationGetHttpItemsList500ApplicationProblemPlusJSONResponse{}, nil
	}

	dto := convertHttpReplicationItems(items)
	return ReplicationGetHttpItemsList200JSONResponse(dto), nil
}

func (s *HttpServer) ReplicationGetHttpItem(ctx context.Context, request ReplicationGetHttpItemRequestObject) (ReplicationGetHttpItemResponseObject, error) {
	if s.services.HttpReplication == nil {
		return ReplicationGetHttpItem501ApplicationProblemPlusJSONResponse{}, nil
	}

	item, err := s.services.HttpReplication.GetReplicationItem(request.Id)
	if err != nil {
		if errors.Is(err, http_replication.ErrHttpReplicationItemNotFound) {
			return ReplicationGetHttpItem400ApplicationProblemPlusJSONResponse{}, nil
		}

		return ReplicationGetHttpItem500ApplicationProblemPlusJSONResponse{}, nil
	}

	dto := convertHttpReplicationItem(item)
	return ReplicationGetHttpItem200JSONResponse(dto), nil
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
		DestUris:         item.ReplicationConf.Destinations,
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
