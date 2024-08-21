package vault

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/services/components/acme"
)

func BuildAcmeService(conf vault.Acme) (*acme.Service, error) {
	client := getVaultClient(conf.VaultId)
	if client == nil {
		return nil, fmt.Errorf("vault client %q not found", conf.VaultId)
	}

	vaultClient, err := acme.NewVaultClient(client.Client().Logical())
	if err != nil {
		return nil, err
	}

	return acme.NewService(vaultClient, conf)
}
