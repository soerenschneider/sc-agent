package vault

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/services/components/vault_pki_ssh"
)

func BuildSshService(conf vault.SshPki) (*vault_pki_ssh.SshService, error) {
	client := getVaultClient(conf.VaultId)
	if client == nil {
		return nil, fmt.Errorf("vault client %q not found", conf.VaultId)
	}

	//var opts []vault_pki_ssh.VaultSshSignerOpts
	//if len(conf.HostKeys) > 0 {
	//	opts = append(opts, vault_pki_ssh.WithHostCerts(conf.HostKeys))
	//}
	//
	//if len(conf.UserKeys) > 0 {
	//	opts = append(opts, vault_pki_ssh.WithUserCerts(conf.UserKeys))
	//}

	vaultClient, err := vault_pki_ssh.NewVaultSshClient(client.Client().SSH())
	if err != nil {
		return nil, err
	}

	return vault_pki_ssh.NewSshService(vaultClient, conf.UserKeys, conf.HostKeys)
}
