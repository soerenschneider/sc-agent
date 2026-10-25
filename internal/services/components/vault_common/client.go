package vault_common

import (
	"context"
	"errors"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

type VaultCommon struct {
	client *api.Client
	auth   api.AuthMethod
	name   string

	tokenRenewer           *TokenRenewer
	approleSecretIdRotator *ApproleSecretIdRotatorService
	rotationOnly           bool
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

// NewRotationOnlyVaultClient returns a client that only rotates its secret_id. It does not keep a token, the rotator
// logs in on demand.
func NewRotationOnlyVaultClient(auth api.AuthMethod, client *api.Client, approleSecretIdRotator *ApproleSecretIdRotatorService) (*VaultCommon, error) {
	if approleSecretIdRotator == nil {
		return nil, errors.New("empty approle secret_id rotator passed")
	}

	ret, err := NewVaultClient(auth, client, nil, approleSecretIdRotator)
	if err != nil {
		return nil, err
	}
	ret.rotationOnly = true
	return ret, nil
}

// RotationOnly returns whether the client only rotates its secret_id and must not be used otherwise.
func (v *VaultCommon) RotationOnly() bool {
	return v.rotationOnly
}

func (v *VaultCommon) Client() *api.Client {
	return v.client
}

func (v *VaultCommon) Auth() api.AuthMethod {
	return v.auth
}

func (v *VaultCommon) StartTokenRenewer(ctx context.Context) {
	if v.tokenRenewer == nil {
		log.Warn().Str(logComponent, "vault").Str("name", v.name).Msg("Token renewal not enabled on this client")
		return
	}

	v.tokenRenewer.StartTokenRenewal(ctx)
}

// LoggedIn returns a channel that is closed once the client has successfully logged in to Vault.
func (v *VaultCommon) LoggedIn() <-chan struct{} {
	if v.tokenRenewer == nil {
		ret := make(chan struct{})
		close(ret)
		return ret
	}

	return v.tokenRenewer.LoggedIn()
}

func (v *VaultCommon) StartApproleSecretIdRotation(ctx context.Context) {
	if v.approleSecretIdRotator == nil {
		log.Warn().Str(logComponent, "vault").Str("name", v.name).Msg("ApproleSecretIdRotation not enabled on this client")
		return
	}

	v.approleSecretIdRotator.StartSecretIdRotation(ctx)
}
