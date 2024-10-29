package http_server

import (
	"context"
	"errors"
	"reflect"

	"github.com/soerenschneider/sc-agent/internal/domain/secret_replication"
)

func (s *HttpServer) ReplicationGetSecretsItem(ctx context.Context, request ReplicationGetSecretsItemRequestObject) (ReplicationGetSecretsItemResponseObject, error) {
	if s.services.SecretsReplication == nil {
		return ReplicationGetSecretsItem501ApplicationProblemPlusJSONResponse{}, nil
	}

	item, err := s.services.SecretsReplication.GetReplicationItem(request.Id)
	if err != nil {
		if errors.Is(err, secret_replication.ErrSecretsReplicationItemNotFound) {
			return ReplicationGetSecretsItem404ApplicationProblemPlusJSONResponse{}, nil
		}

		return ReplicationGetSecretsItem500ApplicationProblemPlusJSONResponse{}, nil
	}

	dto := convertSecretReplicationItem(item)
	return ReplicationGetSecretsItem200JSONResponse{
		DestUri:    dto.DestUri,
		Formatter:  dto.Formatter,
		Id:         dto.Id,
		SecretPath: dto.SecretPath,
		Status:     dto.Status,
	}, nil
}

func (s *HttpServer) ReplicationPostSecretsRequests(ctx context.Context, request ReplicationPostSecretsRequestsRequestObject) (ReplicationPostSecretsRequestsResponseObject, error) {
	if s.services.SecretsReplication == nil {
		return ReplicationPostSecretsRequests501ApplicationProblemPlusJSONResponse{}, nil
	}

	syncSecretRequest, err := s.services.SecretsReplication.GetReplicationItem(request.Params.SecretPath)
	if err != nil {
		if errors.Is(err, secret_replication.ErrSecretsReplicationItemNotFound) {
			return ReplicationPostSecretsRequests404ApplicationProblemPlusJSONResponse{}, nil
		}

		return ReplicationPostSecretsRequests500ApplicationProblemPlusJSONResponse{}, nil
	}

	updatedSecret, err := s.services.SecretsReplication.Replicate(ctx, syncSecretRequest)
	if err != nil {
		return ReplicationPostSecretsRequests500ApplicationProblemPlusJSONResponse{}, nil
	}

	if updatedSecret {
		return ReplicationPostSecretsRequests201Response{}, nil
	}

	return ReplicationPostSecretsRequests200Response{}, nil
}

func (s *HttpServer) ReplicationGetSecretsItemsList(ctx context.Context, request ReplicationGetSecretsItemsListRequestObject) (ReplicationGetSecretsItemsListResponseObject, error) {
	if s.services.SecretsReplication == nil {
		return ReplicationGetSecretsItemsList501ApplicationProblemPlusJSONResponse{}, nil
	}

	configuredSyncItems := s.services.SecretsReplication.GetReplicationItems()
	return ReplicationGetSecretsItemsList200JSONResponse{
		Data: convertSecretReplicationItems(configuredSyncItems).Data,
	}, nil
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
