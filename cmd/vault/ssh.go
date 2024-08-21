package vault

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/soerenschneider/sc-agent/internal/config/vault"
	domain "github.com/soerenschneider/sc-agent/internal/domain/ssh"
	"github.com/soerenschneider/sc-agent/internal/services/components/ssh"
)

func BuildSshService(conf vault.SshPki) (*ssh.Service, error) {
	client := getVaultClient(conf.VaultId)
	if client == nil {
		return nil, fmt.Errorf("vault client %q not found", conf.VaultId)
	}

	wrapper := &VaultWrapper{
		client:    client.Client(),
		mountPath: conf.MountPath,
	}
	vaultClient, err := ssh.NewVaultClient(wrapper, conf.MountPath)
	if err != nil {
		return nil, err
	}

	var managedKeys = map[string]domain.ManagedCertificateConfig{}
	for _, conf := range conf.ManagedKeys {
		managedKeys[conf.Id] = conf.ToDomainModel()
	}
	return ssh.NewService(vaultClient, managedKeys)
}

type VaultWrapper struct {
	client    *api.Client
	mountPath string
}

func (v *VaultWrapper) SignKeyWithContext(ctx context.Context, role string, reqData map[string]any) (*api.Secret, error) {
	return v.client.SSHWithMountPoint(v.mountPath).SignKeyWithContext(ctx, role, reqData)
}

func (v *VaultWrapper) ReadRawWithContext(ctx context.Context, path string) (*api.Response, error) {
	return v.client.Logical().ReadRawWithContext(ctx, path)
}
