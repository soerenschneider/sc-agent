package auth

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/api"
)

type TokenAuth struct {
	token string
}

func NewTokenAuth(token string) (*TokenAuth, error) {
	return &TokenAuth{token}, nil
}

// Login verifies the token against Vault and returns its lifecycle information, so an unreachable Vault or an
// invalid token is detected and renewable tokens can be renewed.
func (t *TokenAuth) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	client.SetToken(t.token)
	lookup, err := client.Auth().Token().LookupSelfWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not lookup token: %w", err)
	}

	renewable, err := lookup.TokenIsRenewable()
	if err != nil {
		return nil, err
	}

	ttl, err := lookup.TokenTTL()
	if err != nil {
		return nil, err
	}

	ret := &api.Secret{
		Auth: &api.SecretAuth{
			ClientToken:   t.token,
			Renewable:     renewable,
			LeaseDuration: int(ttl.Seconds()),
		},
	}

	return ret, nil
}
