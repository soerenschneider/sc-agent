package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

type AppRoleAuth struct {
	mountPath    string
	roleID       string
	secretID     string
	secretIDFile string
	secretIDEnv  string
	unwrap       bool
	mutex        sync.Mutex
	once         sync.Once
}

var _ api.AuthMethod = (*AppRoleAuth)(nil)

// SecretID is a struct that allows you to specify where your application is
// storing the secret ID required for login to the AppRole auth method. The
// recommended secure pattern is to use response-wrapping tokens rather than
// a plaintext value, by passing WithWrappingToken() to NewAppRoleAuth.
// https://learn.hashicorp.com/tutorials/vault/approle-best-practices?in=vault/auth-methods#secretid-delivery-best-practices
type SecretID struct {
	// Path on the file system where the secret ID can be found.
	FromFile string
	// The name of the environment variable containing the application's
	// secret ID.
	FromEnv string
	// The secret ID as a plaintext string value.
	FromString string
}

type LoginOption func(a *AppRoleAuth) error

const (
	defaultMountPath = "approle"
)

// NewAppRoleAuth initializes a new AppRole auth method interface to be
// passed as a parameter to the client.Auth().Login method.
//
// For a secret ID, the recommended secure pattern is to unwrap a one-time-use
// response-wrapping token that was placed here by a trusted orchestrator
// (https://learn.hashicorp.com/tutorials/vault/approle-best-practices?in=vault/auth-methods#secretid-delivery-best-practices)
// To indicate that the filepath points to this wrapping token and not just
// a plaintext secret ID, initialize NewAppRoleAuth with the
// WithWrappingToken LoginOption.
//
// Supported options: WithMountPath, WithWrappingToken
func NewAppRoleAuth(roleID string, secretID *SecretID, opts ...LoginOption) (*AppRoleAuth, error) {
	if roleID == "" {
		return nil, fmt.Errorf("no role ID provided for login")
	}

	if secretID == nil {
		return nil, fmt.Errorf("no secret ID provided for login")
	}

	err := secretID.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid secret ID: %w", err)
	}

	a := &AppRoleAuth{
		mountPath: defaultMountPath,
		roleID:    roleID,
	}

	// secret ID will be read in at login time if it comes from a file or environment variable, in case the underlying value changes
	if secretID.FromFile != "" {
		a.secretIDFile = secretID.FromFile
	}

	if secretID.FromEnv != "" {
		a.secretIDEnv = secretID.FromEnv
	}

	if secretID.FromString != "" {
		a.secretID = secretID.FromString
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		// *AppRoleAuth as the argument
		err := opt(a)
		if err != nil {
			return nil, fmt.Errorf("error with login option: %w", err)
		}
	}

	// return the modified auth struct instance
	return a, nil
}

func (a *AppRoleAuth) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}

	var secretIDValue string
	switch {
	case a.secretID != "":
		secretIDValue = a.secretID
	case a.secretIDFile != "":
		s, err := a.readSecretIDFromFile()
		if err != nil {
			return nil, fmt.Errorf("error reading secret ID: %w", err)
		}
		secretIDValue = s
	case a.secretIDEnv != "":
		s := os.Getenv(a.secretIDEnv)
		if s == "" {
			return nil, fmt.Errorf("secret ID was specified with an environment variable %q with an empty value", a.secretIDEnv)
		}
		secretIDValue = s
	}

	var unwrapErr error
	a.once.Do(func() {
		// if the caller indicated that the value was actually a wrapping token, unwrap it first
		if a.unwrap {
			log.Info().Str("subcomponent", "auth-approle").Str("component", "vault").Msg("attempting to unwrap secret ID")
			unwrappedToken, err := client.Logical().UnwrapWithContext(ctx, secretIDValue)
			if err != nil {
				unwrapErr = fmt.Errorf("unable to unwrap response wrapping token: %w", err)
				return
			}
			log.Info().Str("subcomponent", "auth-approle").Str("component", "vault").Msg("secret_id unwrapped successfully")

			data, ok := unwrappedToken.Data["secret_id"]
			if !ok {
				unwrapErr = fmt.Errorf("unable to unwrap response wrapping token: secret_id not found in response")
				return
			}

			secretIDValue = data.(string)
			// write back unwrapped secret_id to file so it can be consumed by multiple clients
			if a.secretIDFile != "" {
				log.Info().Str("component", "vault").Str("secret_id_file", a.secretIDFile).Msg("Writing unwrapped secret_id to file")
				data := []byte(secretIDValue)
				if err := os.WriteFile(a.secretIDFile, data, 0600); err != nil {
					log.Error().Str("component", "vault").Err(err).Msg("could not write unwrapped secret_id to file")
				}
			} else if a.secretIDEnv != "" {
				// store back unwrapped secret_id so it can be consumed by multiple clients
				a.secretID = secretIDValue
			}
		}
	})

	if unwrapErr != nil {
		return nil, unwrapErr
	}

	loginData := map[string]interface{}{
		"role_id": a.roleID,
		"secret_id": secretIDValue,
	}

	path := fmt.Sprintf("auth/%s/login", a.mountPath)
	resp, err := client.Logical().WriteWithContext(ctx, path, loginData)
	if err != nil {
		return nil, fmt.Errorf("unable to log in with app role auth: %w", err)
	}

	return resp, nil
}

func WithMountPath(mountPath string) LoginOption {
	return func(a *AppRoleAuth) error {
		a.mountPath = mountPath
		return nil
	}
}

func WithWrappingToken() LoginOption {
	return func(a *AppRoleAuth) error {
		a.unwrap = true
		return nil
	}
}

func (a *AppRoleAuth) readSecretIDFromFile() (string, error) {
	secretIDFile, err := os.Open(a.secretIDFile)
	if err != nil {
		return "", fmt.Errorf("unable to open file containing secret ID: %w", err)
	}
	defer secretIDFile.Close()

	limitedReader := io.LimitReader(secretIDFile, 1000)
	secretIDBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("unable to read secret ID: %w", err)
	}

	secretIDValue := strings.TrimSuffix(string(secretIDBytes), "\n")

	return secretIDValue, nil
}

func (secretID *SecretID) validate() error {
	if secretID.FromFile == "" && secretID.FromEnv == "" && secretID.FromString == "" {
		return fmt.Errorf("secret ID for AppRole must be provided with a source file, environment variable, or plaintext string")
	}

	if secretID.FromFile != "" {
		if secretID.FromEnv != "" || secretID.FromString != "" {
			return fmt.Errorf("only one source for the secret ID should be specified")
		}
	}

	if secretID.FromEnv != "" {
		if secretID.FromFile != "" || secretID.FromString != "" {
			return fmt.Errorf("only one source for the secret ID should be specified")
		}
	}

	if secretID.FromString != "" {
		if secretID.FromFile != "" || secretID.FromEnv != "" {
			return fmt.Errorf("only one source for the secret ID should be specified")
		}
	}
	return nil
}
