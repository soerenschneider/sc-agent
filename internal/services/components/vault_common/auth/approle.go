package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/fsutil"
	"github.com/spf13/afero"
)

type AppRoleAuth struct {
	mountPath    string
	roleID       string
	secretID     string
	secretIDFile string
	// secretIDFileOwner is the user the secret_id file is written for, the existing file's owner is kept if empty
	secretIDFileOwner string
	secretIDEnv       string
	unwrap            bool
	// unwrapped is set once the wrapping token was successfully unwrapped, so failed attempts are retried on the
	// next login
	unwrapped    bool
	bootstrapper *SecretIdBootstrapper
	mutex        sync.Mutex
}

// errSecretIDUnusable indicates that the secret_id is missing or not valid anymore and a new one is needed.
var errSecretIDUnusable = errors.New("secret_id unusable")

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
// Supported options: WithMountPath, WithWrappingToken, WithBootstrapper
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

	secretIDValue, err := a.readSecretID(ctx, client)
	if err == nil {
		var resp *api.Secret
		resp, err = a.login(ctx, client, secretIDValue)
		if err == nil || !isInvalidSecretID(err) {
			return resp, err
		}
	}

	if a.bootstrapper == nil || !errors.Is(err, errSecretIDUnusable) {
		return nil, err
	}

	wrappingToken, bootstrapErr := a.bootstrapper.WrappedSecretId(ctx, client)
	if bootstrapErr != nil {
		return nil, fmt.Errorf("%w, bootstrapping secret_id failed: %w", err, bootstrapErr)
	}
	log.Info().Str("subcomponent", "auth-approle").Str("component", "vault").Err(err).Msg("bootstrapped new secret_id as the configured one is not usable")

	secretIDValue, err = a.unwrapSecretID(ctx, client, wrappingToken)
	if err != nil {
		return nil, err
	}

	resp, err := a.login(ctx, client, secretIDValue)
	if err != nil {
		return nil, err
	}
	a.bootstrapper.Reset()
	return resp, nil
}

// readSecretID returns the configured secret_id, unwrapping it first if it was configured to be a wrapping token.
// Errors that indicate that a new secret_id is needed wrap errSecretIDUnusable.
func (a *AppRoleAuth) readSecretID(ctx context.Context, client *api.Client) (string, error) {
	var secretIDValue string
	switch {
	case a.secretID != "":
		secretIDValue = a.secretID
	case a.secretIDFile != "":
		s, err := a.readSecretIDFromFile()
		if err != nil {
			return "", fmt.Errorf("%w: error reading secret ID: %w", errSecretIDUnusable, err)
		}
		if s == "" {
			return "", fmt.Errorf("%w: secret ID file %q is empty", errSecretIDUnusable, a.secretIDFile)
		}
		secretIDValue = s
	case a.secretIDEnv != "":
		s := os.Getenv(a.secretIDEnv)
		if s == "" {
			return "", fmt.Errorf("%w: secret ID was specified with an environment variable %q with an empty value", errSecretIDUnusable, a.secretIDEnv)
		}
		secretIDValue = s
	}

	// if the caller indicated that the value was actually a wrapping token, unwrap it first
	if a.unwrap && !a.unwrapped {
		return a.unwrapSecretID(ctx, client, secretIDValue)
	}

	return secretIDValue, nil
}

// unwrapSecretID unwraps the secret_id and writes it back, so it is used for subsequent logins.
func (a *AppRoleAuth) unwrapSecretID(ctx context.Context, client *api.Client, wrappingToken string) (string, error) {
	log.Info().Str("subcomponent", "auth-approle").Str("component", "vault").Msg("attempting to unwrap secret ID")
	// unwrap using a client without token: the wrapping token is sent as client token and must neither be mixed up
	// with a (possibly expired) client token nor leak into the client
	unwrapClient, err := tokenlessClient(client)
	if err != nil {
		return "", err
	}
	unwrappedToken, err := unwrapClient.Logical().UnwrapWithContext(ctx, wrappingToken)
	if err != nil {
		// a 4xx response means the wrapping token is invalid, expired or has already been used
		var respErr *api.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode >= 400 && respErr.StatusCode < 500 {
			err = fmt.Errorf("%w: %w", errSecretIDUnusable, err)
		}
		return "", fmt.Errorf("unable to unwrap response wrapping token: %w", err)
	}
	if unwrappedToken == nil {
		return "", fmt.Errorf("%w: unable to unwrap response wrapping token: empty response", errSecretIDUnusable)
	}
	log.Info().Str("subcomponent", "auth-approle").Str("component", "vault").Msg("secret_id unwrapped successfully")

	secretIDValue, ok := unwrappedToken.Data["secret_id"].(string)
	if !ok {
		return "", fmt.Errorf("unable to unwrap response wrapping token: secret_id not found in response")
	}

	// write back unwrapped secret_id to file so it can be consumed by multiple clients. Failing to do so is an error,
	// otherwise the secret_id would be lost and a new one would be needed for the next login.
	if a.secretIDFile != "" {
		log.Info().Str("component", "vault").Str("secret_id_file", a.secretIDFile).Msg("Writing unwrapped secret_id to file")
		data := []byte(secretIDValue)
		if err := fsutil.WriteFileAtomic(afero.NewOsFs(), a.secretIDFile, data, a.secretIDFileOwner); err != nil {
			return "", fmt.Errorf("could not write unwrapped secret_id to file: %w", err)
		}
	} else if a.secretIDEnv != "" {
		// store back unwrapped secret_id so it can be consumed by multiple clients
		a.secretID = secretIDValue
	}
	a.unwrapped = true

	return secretIDValue, nil
}

func (a *AppRoleAuth) login(ctx context.Context, client *api.Client, secretIDValue string) (*api.Secret, error) {
	loginData := map[string]interface{}{
		"role_id":   a.roleID,
		"secret_id": secretIDValue,
	}

	path := fmt.Sprintf("auth/%s/login", a.mountPath)
	resp, err := client.Logical().WriteWithContext(ctx, path, loginData)
	if err != nil {
		if isInvalidSecretID(err) {
			err = fmt.Errorf("%w: %w", errSecretIDUnusable, err)
		}
		return nil, fmt.Errorf("unable to log in with app role auth: %w", err)
	}

	return resp, nil
}

func tokenlessClient(client *api.Client) (*api.Client, error) {
	c, err := client.Clone()
	if err != nil {
		return nil, err
	}
	c.ClearToken()
	return c, nil
}

// isInvalidSecretID reports whether Vault rejected the login because of the secret_id, e.g. because it expired, was
// destroyed or exceeded its number of uses. Vault responds with "invalid secret id" or, in newer versions, "invalid
// role or secret ID".
func isInvalidSecretID(err error) bool {
	var respErr *api.ResponseError
	if !errors.As(err, &respErr) || respErr.StatusCode != http.StatusBadRequest {
		return false
	}
	for _, e := range respErr.Errors {
		if strings.Contains(strings.ToLower(e), "secret id") {
			return true
		}
	}
	return false
}

func WithMountPath(mountPath string) LoginOption {
	return func(a *AppRoleAuth) error {
		a.mountPath = mountPath
		return nil
	}
}

// WithBootstrapper fetches a new wrapped secret_id using the bootstrapper if the secret_id is missing or not valid
// anymore.
func WithBootstrapper(b *SecretIdBootstrapper) LoginOption {
	return func(a *AppRoleAuth) error {
		if b == nil {
			return errors.New("empty bootstrapper passed")
		}
		a.bootstrapper = b
		return nil
	}
}

// WithSecretIDFileOwner sets the user the unwrapped secret_id file is written for.
func WithSecretIDFileOwner(owner string) LoginOption {
	return func(a *AppRoleAuth) error {
		a.secretIDFileOwner = owner
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
