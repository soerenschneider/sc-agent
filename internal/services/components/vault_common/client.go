package vault_common

import (
	"context"
	"errors"
	"sync"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

type VaultCommon struct {
	client *api.Client
	auth   api.AuthMethod
	name   string

	tokenRenewer           *TokenRenewer
	approleSecretIdRotator *ApproleSecretIdRotatorService
}

func NewVaultClient(auth api.AuthMethod, client *api.Client, renewer *TokenRenewer, approleSecretIdRotator *ApproleSecretIdRotatorService) (*VaultCommon, error) {
	if auth == nil {
		return nil, errors.New("empty authmethod passed")
	}

	if client == nil {
		return nil, errors.New("empty client passed")
	}

	return &VaultCommon{
		client: client,
		auth:   auth,

		tokenRenewer:           renewer,
		approleSecretIdRotator: approleSecretIdRotator,
	}, nil
}

func (v *VaultCommon) Client() *api.Client {
	return v.client
}

func (v *VaultCommon) Auth() api.AuthMethod {
	return v.auth
}

func (v *VaultCommon) StartTokenRenewer(ctx context.Context, wg *sync.WaitGroup, vaultFatalError chan error) {
	if v.tokenRenewer == nil {
		log.Warn().Str(logComponent, "vault").Str("name", v.name).Msg("Token renewal not enabled on this client")
		return
	}

	v.tokenRenewer.StartTokenRenewal(ctx, wg, vaultFatalError)
}

func (v *VaultCommon) StartApproleSecretIdRotation(ctx context.Context) {
	if v.approleSecretIdRotator == nil {
		log.Warn().Str(logComponent, "vault").Str("name", v.name).Msg("ApproleSecretIdRotation not enabled on this client")
		return
	}

	v.approleSecretIdRotator.StartSecretIdRotation(ctx)
}
