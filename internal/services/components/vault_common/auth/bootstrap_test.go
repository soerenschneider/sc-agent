package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/hashicorp/vault/api"
	"github.com/soerenschneider/sc-agent/internal/fsutil"
)

//nolint:gosec // test fixtures, not credentials
const (
	testRoleName      = "demo"
	testMount         = "approle"
	testMachineId     = "node-42"
	testChallenge     = "challenge-token"
	testWrapToken     = "hvs.wrapping-token"
	testNewSecretId   = "new-secret-id"
	testCreationPath  = "auth/approle/role/demo/secret-id"
	invalidSecretIdIs = "invalid role or secret ID"
)

func ageEncrypt(t *testing.T, recipient age.Recipient, plaintext string) string {
	t.Helper()
	var buf bytes.Buffer
	aw := armor.NewWriter(&buf)
	w, err := age.Encrypt(aw, recipient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, plaintext); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := aw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// fakeApproleCourier issues age-encrypted challenges and wrapping tokens like approle-courier.
type fakeApproleCourier struct {
	recipient age.Recipient
	t         *testing.T

	// challengeStatus and secretIdStatus make the endpoints fail with the given status
	challengeStatus int
	secretIdStatus  int
	// secretIdDelay delays the response after the secret_id was created
	secretIdDelay time.Duration

	mutex      sync.Mutex
	challenges int
	secretIds  int
}

func (f *fakeApproleCourier) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	switch r.URL.Path {
	case "/v1/auth/challenge":
		if f.challengeStatus != 0 {
			writeJSON(w, f.challengeStatus, map[string]any{"errors": []string{"challenge failed"}})
			return
		}
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["machine_id"] != testMachineId {
			writeJSON(w, http.StatusNotFound, map[string]any{"errors": []string{"unknown machine"}})
			return
		}
		f.challenges++
		writeJSON(w, http.StatusCreated, map[string]any{
			"challenge": map[string]string{"format": "age-armor", "value": ageEncrypt(f.t, f.recipient, testChallenge)},
		})
	case "/v1/approle/roles/" + testRoleName + "/secret-id":
		if r.Header.Get("Authorization") != "Bearer "+testChallenge {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"errors": []string{"invalid or expired challenge token"}})
			return
		}
		if r.URL.Query().Get("mount") != testMount {
			writeJSON(w, http.StatusForbidden, map[string]any{"errors": []string{"machine is not authorized for this role"}})
			return
		}
		if f.secretIdStatus != 0 {
			writeJSON(w, f.secretIdStatus, map[string]any{"errors": []string{"secret_id failed"}})
			return
		}
		f.secretIds++
		time.Sleep(f.secretIdDelay)
		writeJSON(w, http.StatusCreated, map[string]any{
			"encrypted_wrap_token": map[string]string{"format": "age-armor", "value": ageEncrypt(f.t, f.recipient, testWrapToken)},
		})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// fakeVault implements the wrapping and approle login endpoints.
type fakeVault struct {
	creationPath string
	loginStatus  int

	mutex     sync.Mutex
	unwrapped bool
	secretIds map[string]bool
}

func (f *fakeVault) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)

	switch r.URL.Path {
	case "/v1/sys/wrapping/lookup":
		if body["token"] != testWrapToken {
			writeJSON(w, http.StatusBadRequest, map[string]any{"errors": []string{"wrapping token is not valid or does not exist"}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"creation_path": f.creationPath}})
	case "/v1/sys/wrapping/unwrap":
		token := r.Header.Get("X-Vault-Token")
		if token == "" {
			token = body["token"]
		}
		if token != testWrapToken || f.unwrapped {
			writeJSON(w, http.StatusBadRequest, map[string]any{"errors": []string{"wrapping token is not valid or does not exist"}})
			return
		}
		f.unwrapped = true
		f.secretIds[testNewSecretId] = true
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"secret_id": testNewSecretId}})
	case "/v1/auth/approle/login":
		if f.loginStatus != 0 {
			writeJSON(w, f.loginStatus, map[string]any{"errors": []string{"internal error"}})
			return
		}
		if !f.secretIds[body["secret_id"]] {
			writeJSON(w, http.StatusBadRequest, map[string]any{"errors": []string{invalidSecretIdIs}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"auth": map[string]any{"client_token": "client-token"}})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type testEnv struct {
	bootstrap    *fakeApproleCourier
	vault        *fakeVault
	client       *api.Client
	bootstrapper *SecretIdBootstrapper
	secretIdFile string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	identityFile := filepath.Join(dir, "age.key")
	if err := os.WriteFile(identityFile, []byte(identity.String()+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	env := &testEnv{
		bootstrap:    &fakeApproleCourier{recipient: identity.Recipient(), t: t},
		vault:        &fakeVault{creationPath: testCreationPath, secretIds: map[string]bool{}},
		secretIdFile: filepath.Join(dir, "secret-id"),
	}

	bootstrapServer := httptest.NewServer(env.bootstrap)
	t.Cleanup(bootstrapServer.Close)
	vaultServer := httptest.NewServer(env.vault)
	t.Cleanup(vaultServer.Close)

	conf := api.DefaultConfig()
	conf.Address = vaultServer.URL
	conf.MaxRetries = 0
	env.client, err = api.NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}

	env.bootstrapper, err = NewSecretIdBootstrapper(bootstrapServer.URL, testMachineId, testMount, testRoleName, identityFile)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func (e *testEnv) appRoleAuth(t *testing.T, opts ...LoginOption) *AppRoleAuth {
	t.Helper()
	opts = append(opts, WithBootstrapper(e.bootstrapper))
	a, err := NewAppRoleAuth("role-id", &SecretID{FromFile: e.secretIdFile}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func (e *testEnv) assertSecretIdFile(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(e.secretIdFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != testNewSecretId {
		t.Fatalf("expected secret_id file to contain %q, got %q", testNewSecretId, data)
	}
}

func TestLogin_BootstrapsMissingSecretIdFile(t *testing.T) {
	env := newTestEnv(t)
	a := env.appRoleAuth(t)

	resp, err := a.Login(context.Background(), env.client)
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if resp.Auth.ClientToken != "client-token" {
		t.Fatalf("unexpected client token %q", resp.Auth.ClientToken)
	}
	env.assertSecretIdFile(t)

	// subsequent logins use the bootstrapped secret_id from the file
	if _, err := a.Login(context.Background(), env.client); err != nil {
		t.Fatalf("expected second login to succeed, got %v", err)
	}
	if env.bootstrap.secretIds != 1 {
		t.Fatalf("expected exactly one bootstrapped secret_id, got %d", env.bootstrap.secretIds)
	}
}

func TestLogin_BootstrapsInvalidSecretId(t *testing.T) {
	env := newTestEnv(t)
	if err := os.WriteFile(env.secretIdFile, []byte("expired-secret-id"), 0600); err != nil {
		t.Fatal(err)
	}
	a := env.appRoleAuth(t)

	if _, err := a.Login(context.Background(), env.client); err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	env.assertSecretIdFile(t)
}

func TestLogin_BootstrapsUsedWrappingToken(t *testing.T) {
	env := newTestEnv(t)
	if err := os.WriteFile(env.secretIdFile, []byte("hvs.already-used"), 0600); err != nil {
		t.Fatal(err)
	}
	a := env.appRoleAuth(t, WithWrappingToken())

	if _, err := a.Login(context.Background(), env.client); err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	env.assertSecretIdFile(t)
}

func TestLogin_NoBootstrapOnServerErrors(t *testing.T) {
	env := newTestEnv(t)
	env.vault.loginStatus = http.StatusInternalServerError
	if err := os.WriteFile(env.secretIdFile, []byte("some-secret-id"), 0600); err != nil {
		t.Fatal(err)
	}
	a := env.appRoleAuth(t)

	if _, err := a.Login(context.Background(), env.client); err == nil {
		t.Fatal("expected login to fail")
	}
	if env.bootstrap.challenges != 0 {
		t.Fatalf("expected no bootstrap attempt, got %d challenges", env.bootstrap.challenges)
	}
}

func TestLogin_WithoutBootstrapperFailsOnMissingFile(t *testing.T) {
	env := newTestEnv(t)
	a, err := NewAppRoleAuth("role-id", &SecretID{FromFile: env.secretIdFile})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := a.Login(context.Background(), env.client); err == nil {
		t.Fatal("expected login to fail")
	}
}

func TestWrappedSecretId_RejectsUnexpectedCreationPath(t *testing.T) {
	env := newTestEnv(t)
	env.vault.creationPath = "secret/data/something-else"

	_, err := env.bootstrapper.WrappedSecretId(context.Background(), env.client)
	if err == nil {
		t.Fatal("expected unexpected creation_path to be rejected")
	}
	if env.vault.unwrapped {
		t.Fatal("wrapping token must not be unwrapped")
	}
}

func TestWrappedSecretId_UnknownMachine(t *testing.T) {
	env := newTestEnv(t)
	env.bootstrapper.machineId = "unknown"

	if _, err := env.bootstrapper.WrappedSecretId(context.Background(), env.client); err == nil {
		t.Fatal("expected unknown machine to fail")
	}
}

func TestWrappedSecretId_Backoff(t *testing.T) {
	env := newTestEnv(t)

	if _, err := env.bootstrapper.WrappedSecretId(context.Background(), env.client); err != nil {
		t.Fatalf("expected first attempt to succeed, got %v", err)
	}
	if _, err := env.bootstrapper.WrappedSecretId(context.Background(), env.client); !errors.Is(err, ErrBootstrapBackoff) {
		t.Fatalf("expected ErrBootstrapBackoff, got %v", err)
	}
	if env.bootstrap.secretIds != 1 {
		t.Fatalf("expected one secret_id request, got %d", env.bootstrap.secretIds)
	}

	env.bootstrapper.Reset()
	if _, err := env.bootstrapper.WrappedSecretId(context.Background(), env.client); err != nil {
		t.Fatalf("expected attempt after reset to succeed, got %v", err)
	}
}

func TestWrappedSecretId_DelayAfterFailure(t *testing.T) {
	tests := []struct {
		name  string
		setup func(env *testEnv)
		// retrySoon is set if the failure is expected to be retried after the short retry delay
		retrySoon bool
	}{
		{
			name: "approle-courier unreachable",
			setup: func(env *testEnv) {
				server := httptest.NewServer(http.NotFoundHandler())
				server.Close()
				env.bootstrapper.address = server.URL
			},
			retrySoon: true,
		},
		{
			name:      "challenge server error",
			setup:     func(env *testEnv) { env.bootstrap.challengeStatus = http.StatusServiceUnavailable },
			retrySoon: true,
		},
		{
			name:      "challenge rate limited",
			setup:     func(env *testEnv) { env.bootstrap.challengeStatus = http.StatusTooManyRequests },
			retrySoon: true,
		},
		{
			name:  "unknown machine",
			setup: func(env *testEnv) { env.bootstrapper.machineId = "unknown" },
		},
		{
			name:      "secret_id server error",
			setup:     func(env *testEnv) { env.bootstrap.secretIdStatus = http.StatusBadGateway },
			retrySoon: true,
		},
		{
			name:  "secret_id forbidden",
			setup: func(env *testEnv) { env.bootstrap.secretIdStatus = http.StatusForbidden },
		},
		{
			name: "secret_id response lost",
			setup: func(env *testEnv) {
				env.bootstrap.secretIdDelay = 200 * time.Millisecond
				env.bootstrapper.httpClient.Timeout = 50 * time.Millisecond
			},
		},
		{
			name:  "unexpected creation_path",
			setup: func(env *testEnv) { env.vault.creationPath = "secret/data/something-else" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			now := time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC)
			env.bootstrapper.now = func() time.Time { return now }
			tt.setup(env)

			if _, err := env.bootstrapper.WrappedSecretId(context.Background(), env.client); err == nil {
				t.Fatal("expected bootstrapping to fail")
			}

			wantDelay, wantBackoff, wantRetryDelay := bootstrapMinBackoff, 2*bootstrapMinBackoff, bootstrapMinRetryDelay
			if tt.retrySoon {
				wantDelay, wantBackoff, wantRetryDelay = bootstrapMinRetryDelay, bootstrapMinBackoff, 2*bootstrapMinRetryDelay
			}
			if got := env.bootstrapper.nextAttempt.Sub(now); got != wantDelay {
				t.Errorf("expected next attempt in %v, got %v", wantDelay, got)
			}
			if env.bootstrapper.backoff != wantBackoff {
				t.Errorf("expected backoff %v, got %v", wantBackoff, env.bootstrapper.backoff)
			}
			if env.bootstrapper.retryDelay != wantRetryDelay {
				t.Errorf("expected retry delay %v, got %v", wantRetryDelay, env.bootstrapper.retryDelay)
			}

			_, err := env.bootstrapper.WrappedSecretId(context.Background(), env.client)
			if !errors.Is(err, ErrBootstrapBackoff) || !strings.Contains(err.Error(), "last attempt failed") {
				t.Fatalf("expected ErrBootstrapBackoff including the last error, got %v", err)
			}
		})
	}
}

func TestLogin_FailsIfSecretIdFileIsNotWritable(t *testing.T) {
	env := newTestEnv(t)
	env.secretIdFile = filepath.Join(t.TempDir(), "missing-dir", "secret-id")
	a := env.appRoleAuth(t)

	if _, err := a.Login(context.Background(), env.client); err == nil {
		t.Fatal("expected login to fail")
	}
	if _, err := a.Login(context.Background(), env.client); !errors.Is(err, ErrBootstrapBackoff) {
		t.Fatalf("expected ErrBootstrapBackoff, got %v", err)
	}
	if env.bootstrap.secretIds != 1 {
		t.Fatalf("expected exactly one bootstrapped secret_id, got %d", env.bootstrap.secretIds)
	}
}

func TestLogin_WritesSecretIdFileForOwner(t *testing.T) {
	env := newTestEnv(t)
	a := env.appRoleAuth(t, WithSecretIDFileOwner("sc-agent-no-such-user"))

	_, err := a.Login(context.Background(), env.client)
	if !errors.Is(err, fsutil.ErrFileOwnership) {
		t.Fatalf("expected the configured owner to be applied, got %v", err)
	}
	if _, err := os.Stat(env.secretIdFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no secret_id file without the configured owner, got %v", err)
	}
}
