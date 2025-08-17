package vault

import (
	"errors"
	"fmt"
	"os"

	"github.com/soerenschneider/sc-agent/internal/config/vault"
	domain "github.com/soerenschneider/sc-agent/internal/domain/secret_replication"
	"github.com/soerenschneider/sc-agent/internal/services/components/secret_replication"
	"github.com/soerenschneider/sc-agent/internal/services/components/secret_replication/formatter"
	"github.com/soerenschneider/sc-agent/internal/storage"
)

const (
	vaultSecretSyncerFormatterYamlKey                = "yaml"
	vaultSecretSyncerFormatterTemplateKey            = "template"
	vaultSecretSyncerFormatterTemplateOptionFile     = "file"
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
			return nil, fmt.Errorf("could not build formatter for %q: %w", req.Formatter, err)
		}

		storageImpl, err := storage.NewFilesystemStorageFromUri(req.DestUri)
		if err != nil {
			return nil, fmt.Errorf("could not build storage for %q: %w", req.DestUri, err)
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
	case vaultSecretSyncerFormatterTemplateKey:
		if arguments == nil {
			return nil, errors.New("no arguments provided")
		}

		var templateData string

		val, found := arguments[vaultSecretSyncerFormatterTemplateOptionFile]
		if !found {
			return nil, errors.New("no 'file' argument provided")
		}

		templateFile, _ := val.(string)
		var err error
		templateData, err = os.Readlink(templateFile)
		if err != nil {
			return nil, fmt.Errorf("could not read template file %q: %w", templateFile, err)
		}

		return formatter.NewTemplateFormatter(templateData)
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
