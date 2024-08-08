package vault

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/services/components/vault_pki_x509"
)

func BuildX509Service(conf vault.X509Pki) (*vault_x509.X509Service, error) {
	client := getVaultClient(conf.VaultId)
	if client == nil {
		return nil, fmt.Errorf("vault client %q not found", conf.VaultId)
	}

	var opts []vault_x509.VaultX509PkiOpts
	vaultX509, err := vault_x509.New(client.Client().Logical(), opts...)
	if err != nil {
		return nil, err
	}

	return vault_x509.NewService(vaultX509, conf)
}
