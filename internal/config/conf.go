package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config/vault"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Http *Http `yaml:"http"`

	Packages           *Packages                 `yaml:"packages"`
	K0s                *K0s                      `yaml:"k0s"`
	PowerStatus        *PowerStatus              `yaml:"power_status"`
	Services           *Services                 `yaml:"services"`
	ReleaseWatcher     *ReleaseWatcher           `yaml:"release_watcher"`
	Libvirt            *Libvirt                  `yaml:"libvirt"`
	Wol                *Wol                      `yaml:"wol"`
	SecretsReplication *vault.SecretsReplication `yaml:"secrets_replication"`
	SshPki             *vault.SshPki             `yaml:"ssh_pki"`
	X509Pki            *vault.X509Pki            `yaml:"x509_pki"`
	ConditionalReboot  *ConditionalRebootConfig  `yaml:"conditional_reboot"`

	Vault map[string]vault.Vault `yaml:"vault"`

	ConfigDir         string `yaml:"config_dir"`
	MetricsListenAddr string `yaml:"metrics_listen_addr"`
	MetricsDir        string
}

type Http struct {
	TlsCertFile    string   `yaml:"tls_cert_file" validate:"omitempty,file"`
	TlsKeyFile     string   `yaml:"tls_key_file" validate:"omitempty,file"`
	TlsCaFile      string   `yaml:"tls_ca_file" validate:"omitempty,file"`
	TlsClientAuth  bool     `yaml:"tls_client_auth"`
	AllowedUserCns []string `yaml:"allowed_user_cns"`
	AllowedEmails  []string `yaml:"allowed_emails"`

	Address string `yaml:"address" validate:"hostname_port"`
}

type Packages struct {
	Enabled bool `yaml:"enabled"`
	UseSudo bool `yaml:"use_sudo"`
}

type ReleaseWatcher struct {
	Enabled bool `yaml:"enabled"`
}

type K0s struct {
	Enabled bool `yaml:"enabled"`
	UseSudo bool `yaml:"use_sudo"`
}

type PowerStatus struct {
	Enabled bool `yaml:"enabled"`
	UseSudo bool `yaml:"use_sudo"`
}

type Libvirt struct {
	Enabled bool `yaml:"enabled"`
	UseSudo bool `yaml:"use_sudo"`
}

type Wol struct {
	Enabled       bool              `yaml:"enabled"`
	Aliases       map[string]string `yaml:"aliases" validate:"required_if=Enabled true"`
	BroadcastAddr string            `yaml:"broadcast" validate:"required,ip"`
}

type Services struct {
	Enabled bool `yaml:"enabled"`
	UseSudo bool `yaml:"use_sudo"`

	UnitsAllowlist []string `yaml:"units_allowlist"`
	UnitsDenylist  []string `yaml:"units_denylist"`
}

func getDefaultConfig() Config {
	conf := Config{
		MetricsListenAddr: "0.0.0.0:9199",
		ReleaseWatcher:    &ReleaseWatcher{Enabled: true},
	}

	return conf
}

func ReadConfig(confFile string) (*Config, error) {
	conf := getDefaultConfig()

	err := env.Parse(&conf)
	if err != nil {
		return nil, err
	}

	if err := readConfigFile(confFile, &conf); err != nil {
		return nil, err
	}

	if conf.ConfigDir != "" {
		if err := readConfigDir(conf.ConfigDir, &conf); err != nil {
			return nil, err
		}
	}

	return &conf, nil
}

func readConfigFile(confFile string, conf *Config) error {
	expandedConfFile, err := expandPath(confFile)
	if err != nil {
		log.Warn().Err(err).Msg("could not expand path, trying verbatim path")
		expandedConfFile = confFile
	}

	content, err := os.ReadFile(expandedConfFile)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(content, conf)
}

func readConfigDir(confDir string, conf *Config) error {
	expandedConfDir, err := expandPath(confDir)
	if err != nil {
		log.Warn().Err(err).Str("path", confDir).Msg("could not expand path, trying verbatim path")
		expandedConfDir = confDir
	}
	configs, err := readConfigFiles(expandedConfDir)
	if err != nil && os.IsNotExist(err) {
		log.Warn().Err(err).Msg("could not read from conf.d dir")
		return nil
	}

	mergedConfigMap := make(map[string]interface{})
	for _, config := range configs {
		mergeMaps(mergedConfigMap, config)
	}

	if err := convertMapToStruct(mergedConfigMap, &conf); err != nil {
		return err
	}

	return nil
}

func readConfigFiles(dir string) ([]map[string]interface{}, error) {
	var configs []map[string]interface{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
			log.Info().Msgf("reading config file from %q", path)
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var config map[string]interface{}
			if err := yaml.Unmarshal(data, &config); err != nil {
				return err
			}
			configs = append(configs, config)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return configs, nil
}

func mergeMaps(dst, src map[string]interface{}) {
	maps.Copy(dst, src)
}

// convertMapToStruct converts a map to a struct
func convertMapToStruct(m map[string]interface{}, s interface{}) error {
	yamlData, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlData, s)
}

// ExpandPath expands the `~` in a path to the user's home directory.
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[1:]), nil
	}

	return path, nil
}
