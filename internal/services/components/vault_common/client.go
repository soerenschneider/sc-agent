package vault_common

import (
	"context"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

type VaultCommon struct {
	client *api.Client
	auth   api.AuthMethod

	tokenRenewer           *TokenRenewer
	approleSecretIdRotator *ApproleSecretIdRotatorService
}

func NewVaultClient(auth api.AuthMethod, client *api.Client, renewer *TokenRenewer, approleSecretIdRotator *ApproleSecretIdRotatorService) (*VaultCommon, error) {
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

func (v *VaultCommon) StartTokenRenewer(ctx context.Context, vaultAuthSuccess chan bool, vaultFatalError chan error) {
	if v.tokenRenewer == nil {
		log.Warn().Msg("Token renewal not enabled on this client")
		return
	}

	v.tokenRenewer.StartTokenRenewal(ctx, vaultAuthSuccess, vaultFatalError)
}

func (v *VaultCommon) StartApproleSecretIdRotation(ctx context.Context) {
	if v.approleSecretIdRotator == nil {
		log.Warn().Msg("ApproleSecretIdRotation not enabled on this client")
		return
	}

	v.approleSecretIdRotator.StartSecretIdRotation(ctx)
}
