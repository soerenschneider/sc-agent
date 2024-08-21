package vault

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/services/components/pki"
)

func BuildPkiService(conf vault.X509Pki) (*pki.Service, error) {
	client := getVaultClient(conf.VaultId)
	if client == nil {
		return nil, fmt.Errorf("vault client %q not found", conf.VaultId)
	}

	opts := []pki.VaultClientOpts{
		pki.WithMountPath(conf.MountPath),
	}
	vaultClient, err := pki.NewVaultClient(client.Client().Logical(), opts...)
	if err != nil {
		return nil, err
	}

	return pki.NewService(vaultClient, conf)
}
