package vault

import (
	"errors"
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config/vault"
	domain "github.com/soerenschneider/sc-agent/internal/domain/secret_replication"
	"github.com/soerenschneider/sc-agent/internal/services/components/secret_replication"
	"github.com/soerenschneider/sc-agent/internal/services/components/secret_replication/formatter"
	"github.com/soerenschneider/sc-agent/internal/storage"
)

const (
	vaultSecretSyncerFormatterYamlKey                = "yaml"
	vaultSecretSyncerFormatterJsonKey                = "json"
	vaultSecretSyncerFormatterEnvKey                 = "env"
	vaultSecretSyncerFormatterEnvOptionUppercaseKeys = "uppercase_keys"
)

type Formatter interface {
	Format(data map[string]any) ([]byte, error)
}

func BuildSecretReplication(conf *vault.SecretsReplication) (*secret_replication.Service, error) {
	if conf == nil {
		return nil, errors.New("no vaultsyncer config given")
	}

	client := getVaultClient(conf.VaultId)
	if client == nil {
		return nil, fmt.Errorf("vault client %q not found", conf.VaultId)
	}

	kv2Client, err := secret_replication.NewClient(client.Client().KVv2(conf.Kv2Mount))
	if err != nil {
		return nil, err
	}

	syncRequests, err := buildSyncSecretRequests(*conf)
	if err != nil {
		return nil, err
	}

	if len(syncRequests) == 0 {
		return nil, errors.New("secret syncer config provided but no actual secrets to sync defined")
	}

	return secret_replication.NewService(kv2Client, syncRequests)
}

func buildSyncSecretRequests(conf vault.SecretsReplication) ([]domain.ReplicationItem, error) {
	var ret []domain.ReplicationItem

	for id, req := range conf.ReplicationRequests {
		formatter, err := buildSecretFormatter(req.Formatter, req.FormatterArgs)
		if err != nil {
			return nil, err
		}

		storageImpl, err := storage.NewFilesystemStorageFromUri(req.DestUri)
		if err != nil {
			return nil, err
		}

		request := domain.ReplicationItem{
			ReplicationConf: domain.ReplicationConf{
				Id:         id,
				SecretPath: req.SecretPath,
				DestUri:    req.DestUri,
			},
			Formatter:   formatter,
			Destination: storageImpl,
		}

		ret = append(ret, request)
	}

	return ret, nil
}

func buildSecretFormatter(name string, arguments map[string]any) (Formatter, error) {
	switch name {
	case vaultSecretSyncerFormatterEnvKey:
		uppercaseKeys := false
		if arguments != nil {
			val, found := arguments[vaultSecretSyncerFormatterEnvOptionUppercaseKeys]
			if found {
				convertedVal, success := val.(bool)
				if success {
					uppercaseKeys = convertedVal
				}
			}
		}

		return formatter.NewEnvVarFormatter(uppercaseKeys), nil
	case vaultSecretSyncerFormatterYamlKey:
		return &formatter.YamlFormatter{}, nil
	case vaultSecretSyncerFormatterJsonKey:
		return &formatter.JsonFormatter{}, nil
	default:
		return nil, errors.New("no implementation found")
	}
}
