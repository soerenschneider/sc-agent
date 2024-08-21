package ports

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/domain/http_replication"
)

type HttpReplication interface {
	Replicate(ctx context.Context, item http_replication.ReplicationItem) error
	StartReplication(ctx context.Context)
	GetReplicationItem(id string) (http_replication.ReplicationItem, error)
	GetReplicationItems() ([]http_replication.ReplicationItem, error)
}
