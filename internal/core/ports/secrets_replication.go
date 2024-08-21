package ports

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/domain"
)

type SecretsReplication interface {
	// Replicate replicates a secret item based on the provided SecretReplicationItem. It takes a context for cancellation
	// and a SecretReplicationItem which contains the details of the secret to be synchronized.
	//
	// Parameters:
	//   ctx (context.Context): The context for controlling cancellation and timeout.
	//   syncRequest (domain.SecretReplicationItem): The request object containing the details of the secret to be synchronized.
	//
	// Returns:
	//   bool: A boolean value indicating whether the secret was updated on the destination or not.
	//   error: An error object if there was an issue during synchronization, otherwise nil.
	Replicate(ctx context.Context, syncRequest domain.SecretReplicationItem) (bool, error)
	GetReplicationItem(id string) (domain.SecretReplicationItem, error)
	GetReplicationItems() []domain.SecretReplicationItem
	StartContinuousReplication(ctx context.Context)
}
