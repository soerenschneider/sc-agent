package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/hashicorp/vault/api"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

const (
	defaultBootstrapWrapTtl = 2 * time.Minute
	bootstrapHttpTimeout    = 30 * time.Second
	// bootstrapMinBackoff and bootstrapMaxBackoff bound the delay between bootstrap attempts. Every attempt creates a
	// new secret_id in Vault, so it must not run on every (failed) login.
	bootstrapMinBackoff = time.Minute
	bootstrapMaxBackoff = time.Hour
	maxResponseSize     = 64 * 1024
)

var ErrBootstrapBackoff = errors.New("bootstrapping secret_id is backing off")

// SecretIdBootstrapper fetches response-wrapped secret_ids from approle-courier. The machine authenticates using an
// age-encrypted challenge that only the holder of its age identity can decrypt, the returned wrapping token is
// encrypted to the same identity.
type SecretIdBootstrapper struct {
	address    string
	machineId  string
	mount      string
	roleName   string
	wrapTtl    time.Duration
	identities []age.Identity
	httpClient *http.Client
	// metricsLabel is the secret_id file the bootstrapped secret_id is written to
	metricsLabel string

	mutex       sync.Mutex
	backoff     time.Duration
	nextAttempt time.Time
	// lastErr is the error of the last failed attempt, it is returned while backing off so the cause is not lost
	lastErr error
	now     func() time.Time
}

type BootstrapOption func(b *SecretIdBootstrapper) error

func NewSecretIdBootstrapper(address, machineId, mount, roleName, ageIdentityFile string, opts ...BootstrapOption) (*SecretIdBootstrapper, error) {
	if address == "" {
		return nil, errors.New("no approle-courier address provided")
	}
	if machineId == "" {
		return nil, errors.New("no machine id provided")
	}
	if mount == "" || roleName == "" {
		return nil, errors.New("no approle mount or role name provided")
	}

	identities, err := readAgeIdentities(ageIdentityFile)
	if err != nil {
		return nil, err
	}

	b := &SecretIdBootstrapper{
		address:    strings.TrimSuffix(address, "/"),
		machineId:  machineId,
		mount:      mount,
		roleName:   roleName,
		wrapTtl:    defaultBootstrapWrapTtl,
		identities: identities,
		httpClient: &http.Client{Timeout: bootstrapHttpTimeout},
		backoff:    bootstrapMinBackoff,
		now:        time.Now,
	}

	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func WithBootstrapWrapTtl(ttl time.Duration) BootstrapOption {
	return func(b *SecretIdBootstrapper) error {
		if ttl < time.Second {
			return errors.New("wrap_ttl must be at least 1s")
		}
		b.wrapTtl = ttl
		return nil
	}
}

func WithBootstrapMetricsLabel(secretIdFile string) BootstrapOption {
	return func(b *SecretIdBootstrapper) error {
		b.metricsLabel = secretIdFile
		return nil
	}
}

func readAgeIdentities(file string) ([]age.Identity, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("could not open age identity file: %w", err)
	}
	defer f.Close()

	identities, err := age.ParseIdentities(f)
	if err != nil {
		return nil, fmt.Errorf("could not parse age identity file %q: %w", file, err)
	}
	return identities, nil
}

// WrappedSecretId fetches a new response-wrapped secret_id and returns the wrapping token after verifying with Vault
// that it wraps a secret_id of the configured role. Attempts are rate limited with an exponential backoff that is
// reset by Reset, ErrBootstrapBackoff is returned while backing off.
func (b *SecretIdBootstrapper) WrappedSecretId(ctx context.Context, client *api.Client) (string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	now := b.now()
	if now.Before(b.nextAttempt) {
		if b.lastErr != nil {
			return "", fmt.Errorf("%w until %s, last attempt failed: %w", ErrBootstrapBackoff, b.nextAttempt.Format(time.RFC3339), b.lastErr)
		}
		return "", fmt.Errorf("%w until %s", ErrBootstrapBackoff, b.nextAttempt.Format(time.RFC3339))
	}
	b.nextAttempt = now.Add(b.backoff)
	b.backoff = min(2*b.backoff, bootstrapMaxBackoff)

	token, reason, err := b.wrappedSecretId(ctx, client)
	b.lastErr = err
	if err != nil {
		metrics.SecretIdBootstrapErrors.WithLabelValues(b.metricsLabel, reason).Inc()
		return "", err
	}
	metrics.SecretIdBootstraps.WithLabelValues(b.metricsLabel).Inc()
	return token, nil
}

// Reset resets the backoff after the bootstrapped secret_id was used successfully.
func (b *SecretIdBootstrapper) Reset() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.backoff = bootstrapMinBackoff
	b.nextAttempt = time.Time{}
	b.lastErr = nil
}

type ageCiphertext struct {
	Value string `json:"value"`
}

type challengeResponse struct {
	Challenge ageCiphertext `json:"challenge"`
}

type wrappedSecretIdResponse struct {
	EncryptedWrapToken ageCiphertext `json:"encrypted_wrap_token"`
}

type bootstrapErrorResponse struct {
	Errors []string `json:"errors"`
}

// wrappedSecretId returns the wrapping token, or a short reason for the metrics and an error.
func (b *SecretIdBootstrapper) wrappedSecretId(ctx context.Context, client *api.Client) (string, string, error) {
	var challenge challengeResponse
	if err := b.post(ctx, "/v1/auth/challenge", "", nil, map[string]string{"machine_id": b.machineId}, &challenge); err != nil {
		return "", "challenge", fmt.Errorf("could not request challenge: %w", err)
	}
	bearer, err := b.decrypt(challenge.Challenge.Value)
	if err != nil {
		return "", "decrypt_challenge", fmt.Errorf("could not decrypt challenge: %w", err)
	}

	path := fmt.Sprintf("/v1/approle/roles/%s/secret-id", url.PathEscape(b.roleName))
	query := url.Values{"mount": []string{b.mount}}
	body := map[string]any{"wrap_ttl": int(b.wrapTtl.Seconds())}
	var resp wrappedSecretIdResponse
	if err := b.post(ctx, path, bearer, query, body, &resp); err != nil {
		return "", "secret_id", fmt.Errorf("could not request wrapped secret_id: %w", err)
	}
	wrappingToken, err := b.decrypt(resp.EncryptedWrapToken.Value)
	if err != nil {
		return "", "decrypt_wrap_token", fmt.Errorf("could not decrypt wrapping token: %w", err)
	}

	if reason, err := b.verifyWrappingToken(ctx, client, wrappingToken); err != nil {
		return "", reason, err
	}
	return wrappingToken, "", nil
}

// verifyWrappingToken makes sure the wrapping token was issued by Vault for a secret_id of the configured role before
// it is unwrapped, so a compromised approle-courier can not hand out arbitrary wrapped responses.
func (b *SecretIdBootstrapper) verifyWrappingToken(ctx context.Context, client *api.Client, wrappingToken string) (string, error) {
	// sys/wrapping/lookup is unauthenticated, don't send a (possibly expired) client token
	c, err := tokenlessClient(client)
	if err != nil {
		return "lookup", err
	}
	lookup, err := c.Logical().WriteWithContext(ctx, "sys/wrapping/lookup", map[string]any{"token": wrappingToken})
	if err != nil {
		return "lookup", fmt.Errorf("could not lookup wrapping token: %w", err)
	}
	if lookup == nil || lookup.Data == nil {
		return "lookup", errors.New("could not lookup wrapping token: empty response")
	}

	expected := fmt.Sprintf("auth/%s/role/%s/secret-id", b.mount, b.roleName)
	if path, _ := lookup.Data["creation_path"].(string); path != expected {
		return "creation_path", fmt.Errorf("wrapping token has unexpected creation_path %q, expected %q", path, expected)
	}
	return "", nil
}

func (b *SecretIdBootstrapper) post(ctx context.Context, path, bearer string, query url.Values, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	endpoint := b.address + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		var errResp bootstrapErrorResponse
		_ = json.Unmarshal(data, &errResp)
		return fmt.Errorf("approle-courier returned status %d: %s", resp.StatusCode, strings.Join(errResp.Errors, ", "))
	}
	return json.Unmarshal(data, out)
}

func (b *SecretIdBootstrapper) decrypt(armored string) (string, error) {
	r, err := age.Decrypt(armor.NewReader(strings.NewReader(armored)), b.identities...)
	if err != nil {
		return "", err
	}
	plaintext, err := io.ReadAll(io.LimitReader(r, maxResponseSize))
	if err != nil {
		return "", err
	}
	if len(plaintext) == 0 {
		return "", errors.New("empty plaintext")
	}
	return string(plaintext), nil
}
