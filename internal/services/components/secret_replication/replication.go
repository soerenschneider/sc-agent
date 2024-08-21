package secret_replication

import (
	"context"
	"errors"

	vault "github.com/hashicorp/vault/api"
)

type Kv2Client interface {
	Get(ctx context.Context, path string) (*vault.KVSecret, error)
}

type VaultKv2Client struct {
	client Kv2Client
}

func NewClient(client Kv2Client) (*VaultKv2Client, error) {
	if client == nil {
		return nil, errors.New("empty client passed")
	}

	ret := &VaultKv2Client{
		client: client,
	}

	return ret, nil
}

func (s *VaultKv2Client) ReadSecret(ctx context.Context, path string) (*vault.KVSecret, error) {
	return s.client.Get(ctx, path)
}
