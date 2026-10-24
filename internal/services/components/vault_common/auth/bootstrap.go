package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

const (
	defaultBootstrapWrapTtl = 2 * time.Minute
	bootstrapHttpTimeout    = 30 * time.Second
	// bootstrapDialTimeout is shorter than bootstrapHttpTimeout, so an unreachable approle-courier is recognized as
	// such instead of as a request that timed out after it may have been processed.
	bootstrapDialTimeout = 10 * time.Second
	// bootstrapMinBackoff and bootstrapMaxBackoff bound the delay after attempts that may have created a secret_id in
	// Vault or failed permanently, so a broken setup does not flood Vault with secret_ids.
	bootstrapMinBackoff = time.Minute
	bootstrapMaxBackoff = time.Hour
	// bootstrapMinRetryDelay and bootstrapMaxRetryDelay bound the delay after transient failures that happened before
	// a secret_id could have been created, e.g. while approle-courier is unreachable.
	bootstrapMinRetryDelay = time.Minute
	bootstrapMaxRetryDelay = 5 * time.Minute
	maxResponseSize        = 64 * 1024
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
	retryDelay  time.Duration
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
		httpClient: newBootstrapHttpClient(),
		backoff:    bootstrapMinBackoff,
		retryDelay: bootstrapMinRetryDelay,
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

func newBootstrapHttpClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: bootstrapDialTimeout, KeepAlive: 30 * time.Second}).DialContext
	return &http.Client{Timeout: bootstrapHttpTimeout, Transport: transport}
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
// that it wraps a secret_id of the configured role. Attempts are rate limited, ErrBootstrapBackoff is returned while
// backing off. Transient failures that can not have created a secret_id are retried after a short delay, all other
// attempts are followed by an exponential backoff that is reset by Reset.
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

	log.Info().Str("subcomponent", "auth-approle").Str("component", "vault").Msg("bootstrapping new secret_id")
	token, failure, err := b.wrappedSecretId(ctx, client)
	b.lastErr = err
	if err != nil && failure.retrySoon {
		b.nextAttempt = now.Add(b.retryDelay)
		b.retryDelay = min(2*b.retryDelay, bootstrapMaxRetryDelay)
	} else {
		b.nextAttempt = now.Add(b.backoff)
		b.backoff = min(2*b.backoff, bootstrapMaxBackoff)
	}
	if err != nil {
		metrics.SecretIdBootstrapErrors.WithLabelValues(b.metricsLabel, failure.reason).Inc()
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
	b.retryDelay = bootstrapMinRetryDelay
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

// statusError is returned if approle-courier rejected a request.
type statusError struct {
	code   int
	errors []string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("approle-courier returned status %d: %s", e.code, strings.Join(e.errors, ", "))
}

// bootstrapFailure describes a failed bootstrap attempt.
type bootstrapFailure struct {
	// reason is a short description for the metrics
	reason string
	// retrySoon is set if the attempt failed transiently before a secret_id could have been created
	retrySoon bool
}

// wrappedSecretId returns the wrapping token, or a description of the failure and an error.
func (b *SecretIdBootstrapper) wrappedSecretId(ctx context.Context, client *api.Client) (string, bootstrapFailure, error) {
	var challenge challengeResponse
	if err := b.post(ctx, "/v1/auth/challenge", "", nil, map[string]string{"machine_id": b.machineId}, &challenge); err != nil {
		return "", bootstrapFailure{"challenge", isTransient(err, false)}, fmt.Errorf("could not request challenge: %w", err)
	}
	bearer, err := b.decrypt(challenge.Challenge.Value)
	if err != nil {
		return "", bootstrapFailure{reason: "decrypt_challenge"}, fmt.Errorf("could not decrypt challenge: %w", err)
	}

	path := fmt.Sprintf("/v1/approle/roles/%s/secret-id", url.PathEscape(b.roleName))
	query := url.Values{"mount": []string{b.mount}}
	body := map[string]any{"wrap_ttl": int(b.wrapTtl.Seconds())}
	var resp wrappedSecretIdResponse
	if err := b.post(ctx, path, bearer, query, body, &resp); err != nil {
		return "", bootstrapFailure{"secret_id", isTransient(err, true)}, fmt.Errorf("could not request wrapped secret_id: %w", err)
	}
	wrappingToken, err := b.decrypt(resp.EncryptedWrapToken.Value)
	if err != nil {
		return "", bootstrapFailure{reason: "decrypt_wrap_token"}, fmt.Errorf("could not decrypt wrapping token: %w", err)
	}

	if reason, err := b.verifyWrappingToken(ctx, client, wrappingToken); err != nil {
		return "", bootstrapFailure{reason: reason}, err
	}
	return wrappingToken, bootstrapFailure{}, nil
}

// isTransient reports whether a failed request to approle-courier may succeed when retried soon and can not have
// created a secret_id. createsSecretId is set for requests that create a secret_id once approle-courier processes them.
func isTransient(err error, createsSecretId bool) bool {
	// approle-courier rejected the request without creating a secret_id
	var statusErr *statusError
	if errors.As(err, &statusErr) {
		return statusErr.code >= http.StatusInternalServerError || statusErr.code == http.StatusTooManyRequests
	}

	// the request never reached approle-courier
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}

	// the request may have been processed by approle-courier but the response got lost
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return !createsSecretId
	}

	// the response could not be read or parsed
	return false
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
		return &statusError{code: resp.StatusCode, errors: errResp.Errors}
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
