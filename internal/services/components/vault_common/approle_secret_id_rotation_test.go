package vault_common

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/prometheus/client_golang/prometheus/testutil"
	vault_config "github.com/soerenschneider/sc-agent/internal/config/vault"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/pkg/vault"
	"github.com/spf13/afero"
)

// fakeApproleClient returns info as lookup result for the old and newInfo for the new secret_id, or treats the secret_id
// as expired if they are not set. It records the role names it was called with and, if vaultClient is set, the token
// of the Vault client at the time of each call.
type fakeApproleClient struct {
	info        *vault.SecretIdInfo
	newInfo     *vault.SecretIdInfo
	lookupErr   error
	vaultClient *api.Client

	destroyed []string

	roleNames []string
	tokens    []string
	generated int
}

func (f *fakeApproleClient) record(roleName string) {
	f.roleNames = append(f.roleNames, roleName)
	if f.vaultClient != nil {
		f.tokens = append(f.tokens, f.vaultClient.Token())
	}
}

func (f *fakeApproleClient) DestroySecretId(roleName, secretId string, isAccessor bool) error {
	f.record(roleName)
	f.destroyed = append(f.destroyed, secretId)
	return nil
}

func (f *fakeApproleClient) Lookup(roleName, secretId string, isAccessor bool) (*vault.SecretIdInfo, error) {
	f.record(roleName)
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	info := f.info
	if secretId == "new-secret-id" {
		info = f.newInfo
	}
	if info == nil {
		return nil, errAlreadyExpired
	}
	return info, nil
}

func (f *fakeApproleClient) GetSecretIdAccessors(roleName string) ([]string, error) {
	f.roleNames = append(f.roleNames, roleName)
	return nil, nil
}

func (f *fakeApproleClient) GenerateSecretId(roleName string) (string, error) {
	f.record(roleName)
	f.generated++
	return "new-secret-id", nil
}

func (f *fakeApproleClient) ReadRoleId(roleName string) (string, error) {
	f.roleNames = append(f.roleNames, roleName)
	return "", nil
}

func TestConditionallyRotateSecretId_RoleName(t *testing.T) {
	tests := []struct {
		name         string
		conf         vault_config.Vault
		wantRoleName string
	}{
		{
			name:         "role_name configured",
			conf:         vault_config.Vault{RoleId: "3c5e8a0b-1f2d-4e6a-9b7c-8d0e1f2a3b4c", RoleName: "demo", SecretIdFile: "/secret-id"},
			wantRoleName: "demo",
		},
		{
			name:         "defaults to role_id",
			conf:         vault_config.Vault{RoleId: "demo", SecretIdFile: "/secret-id"},
			wantRoleName: "demo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeApproleClient{}
			rotator, err := NewApproleUpdater(client, "default", &tt.conf)
			if err != nil {
				t.Fatal(err)
			}
			rotator.fsImpl = afero.NewMemMapFs()
			if err := afero.WriteFile(rotator.fsImpl, tt.conf.SecretIdFile, []byte("old-secret-id"), 0600); err != nil {
				t.Fatal(err)
			}

			if err := rotator.ConditionallyRotateSecretId(tt.conf, false); err != nil {
				t.Fatalf("expected rotation to succeed, got %v", err)
			}

			if len(client.roleNames) != 4 {
				t.Fatalf("expected lookup, generate, destroy and lookup calls, got %v", client.roleNames)
			}
			for _, roleName := range client.roleNames {
				if roleName != tt.wantRoleName {
					t.Fatalf("expected role name %q, got %v", tt.wantRoleName, client.roleNames)
				}
			}
			data, err := afero.ReadFile(rotator.fsImpl, tt.conf.SecretIdFile)
			if err != nil || string(data) != "new-secret-id" {
				t.Fatalf("expected new secret_id to be written, got %q (%v)", data, err)
			}
		})
	}
}

// fakeAuth hands out a token without talking to Vault, or fails if err is set.
type fakeAuth struct {
	err    error
	logins int
}

func (f *fakeAuth) Login(_ context.Context, _ *api.Client) (*api.Secret, error) {
	f.logins++
	if f.err != nil {
		return nil, f.err
	}
	return &api.Secret{Auth: &api.SecretAuth{ClientToken: "rotation-token"}}, nil
}

// fakeTokenRevoker records the tokens revoked via auth/token/revoke-self.
type fakeTokenRevoker struct {
	mutex   sync.Mutex
	revoked []string
}

func (f *fakeTokenRevoker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if r.URL.Path != "/v1/auth/token/revoke-self" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	f.revoked = append(f.revoked, r.Header.Get("X-Vault-Token"))
	w.WriteHeader(http.StatusNoContent)
}

func TestCheck_OnDemandLogin(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		info         *vault.SecretIdInfo
		newInfo      *vault.SecretIdInfo
		loginErr     error
		wantDelay    time.Duration
		wantRotation bool
		wantCalls    int
	}{
		{
			// 30h lifetime, 20h left: rotation is due at 15h left
			name:      "secret_id still valid",
			info:      &vault.SecretIdInfo{CreationTime: now.Add(-10 * time.Hour), Expiration: now.Add(20 * time.Hour)},
			wantDelay: 5 * time.Hour,
			wantCalls: 1,
		},
		{
			name:      "secret_id rotation due in more than a day",
			info:      &vault.SecretIdInfo{CreationTime: now.Add(-24 * time.Hour), Expiration: now.Add(30 * 24 * time.Hour)},
			wantDelay: maxOnDemandInterval,
			wantCalls: 1,
		},
		{
			name:      "secret_id without expiration",
			info:      &vault.SecretIdInfo{CreationTime: now.Add(-24 * time.Hour)},
			wantDelay: maxOnDemandInterval,
			wantCalls: 1,
		},
		{
			// 30h lifetime, 10h left: rotate, the new secret_id is checked again after half its lifetime
			name:         "secret_id rotated",
			info:         &vault.SecretIdInfo{CreationTime: now.Add(-20 * time.Hour), Expiration: now.Add(10 * time.Hour)},
			newInfo:      &vault.SecretIdInfo{CreationTime: now, Expiration: now.Add(30 * time.Hour)},
			wantDelay:    15 * time.Hour,
			wantRotation: true,
			wantCalls:    4,
		},
		{
			// the new secret_id can not be looked up, it is checked again soon
			name:         "expired secret_id rotated",
			wantDelay:    minOnDemandInterval,
			wantRotation: true,
			wantCalls:    4,
		},
		{
			name:      "login fails",
			loginErr:  errors.New("permission denied"),
			wantDelay: minOnDemandInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revoker := &fakeTokenRevoker{}
			server := httptest.NewServer(revoker)
			defer server.Close()
			conf := api.DefaultConfig()
			conf.Address = server.URL
			vaultClient, err := api.NewClient(conf)
			if err != nil {
				t.Fatal(err)
			}
			vaultClient.ClearToken()

			approleClient := &fakeApproleClient{info: tt.info, newInfo: tt.newInfo, vaultClient: vaultClient}
			auth := &fakeAuth{err: tt.loginErr}
			// unique per test case, the metrics are global
			clientId := strings.ReplaceAll(tt.name, " ", "-")
			vaultConf := &vault_config.Vault{RoleId: "restic", SecretIdFile: "/" + clientId, RotationOnly: true}
			rotator, err := NewApproleUpdater(approleClient, clientId, vaultConf, WithOnDemandLogin(vaultClient, auth))
			if err != nil {
				t.Fatal(err)
			}
			rotator.now = func() time.Time { return now }
			rotator.fsImpl = afero.NewMemMapFs()
			if err := afero.WriteFile(rotator.fsImpl, vaultConf.SecretIdFile, []byte("old-secret-id"), 0600); err != nil {
				t.Fatal(err)
			}

			delay := rotator.check(context.Background())

			// GetPercentage uses the real clock, allow for the time passed since now
			if diff := delay - tt.wantDelay; diff < -time.Second || diff > time.Second {
				t.Errorf("expected next check in %v, got %v", tt.wantDelay, delay)
			}
			if auth.logins != 1 {
				t.Errorf("expected exactly one login, got %d", auth.logins)
			}
			if len(approleClient.tokens) != tt.wantCalls {
				t.Fatalf("expected %d approle calls, got %d", tt.wantCalls, len(approleClient.tokens))
			}
			for _, token := range approleClient.tokens {
				if token != "rotation-token" {
					t.Errorf("expected approle calls to use the token from the login, got %q", token)
				}
			}
			if gotRotation := approleClient.generated > 0; gotRotation != tt.wantRotation {
				t.Errorf("expected rotation %v, got %v", tt.wantRotation, gotRotation)
			}

			wantRevoked := []string{"rotation-token"}
			if tt.loginErr != nil {
				wantRevoked = nil
			}
			if len(revoker.revoked) != len(wantRevoked) || (len(wantRevoked) > 0 && revoker.revoked[0] != wantRevoked[0]) {
				t.Errorf("expected revoked tokens %v, got %v", wantRevoked, revoker.revoked)
			}
			if vaultClient.Token() != "" {
				t.Errorf("expected token to be cleared after the check, got %q", vaultClient.Token())
			}

			wantLogins, wantLoginErrors, wantDegraded := 1., 0., 0.
			if tt.loginErr != nil {
				wantLogins, wantLoginErrors, wantDegraded = 0, 1, 1
			}
			if got := testutil.ToFloat64(metrics.VaultLogins.WithLabelValues(clientId)); got != wantLogins {
				t.Errorf("expected %v logins, got %v", wantLogins, got)
			}
			if got := testutil.ToFloat64(metrics.VaultLoginErrors.WithLabelValues(clientId)); got != wantLoginErrors {
				t.Errorf("expected %v login errors, got %v", wantLoginErrors, got)
			}
			if got := testutil.ToFloat64(metrics.ComponentDegraded.WithLabelValues(DegradedComponentName(clientId))); got != wantDegraded {
				t.Errorf("expected degraded %v, got %v", wantDegraded, got)
			}

			// the expiration of the current secret_id is exported, it is absent if it is unknown or never expires
			wantInfo := tt.info
			if tt.wantRotation {
				wantInfo = tt.newInfo
			}
			wantExpiration := wantInfo != nil && !wantInfo.Expiration.IsZero()
			if wantExpiration {
				if got := testutil.ToFloat64(metrics.SecretIdExpiration.WithLabelValues(vaultConf.SecretIdFile)); got != float64(wantInfo.Expiration.Unix()) {
					t.Errorf("expected expiration %v, got %v", wantInfo.Expiration.Unix(), got)
				}
			}
			// deleting reports whether the series existed
			if got := metrics.SecretIdExpiration.DeleteLabelValues(vaultConf.SecretIdFile); got != wantExpiration {
				t.Errorf("expected expiration series to exist: %v, got %v", wantExpiration, got)
			}
		})
	}
}

func newTestRotator(t *testing.T, client ApproleClient, fs afero.Fs, secretIdFile string) (*ApproleSecretIdRotatorService, vault_config.Vault) {
	t.Helper()
	conf := vault_config.Vault{RoleId: "restic", SecretIdFile: secretIdFile}
	rotator, err := NewApproleUpdater(client, "restic", &conf)
	if err != nil {
		t.Fatal(err)
	}
	rotator.fsImpl = fs
	return rotator, conf
}

func TestConditionallyRotateSecretId_DestroysNewSecretIdIfWriteFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "/destroy-new/secret-id", []byte("old-secret-id"), 0600); err != nil {
		t.Fatal(err)
	}
	client := &fakeApproleClient{}
	rotator, conf := newTestRotator(t, client, afero.NewReadOnlyFs(fs), "/destroy-new/secret-id")

	if err := rotator.ConditionallyRotateSecretId(conf, false); err == nil {
		t.Fatal("expected rotation to fail")
	}
	if client.generated != 1 {
		t.Fatalf("expected a new secret_id to be generated, got %d", client.generated)
	}
	if len(client.destroyed) != 1 || client.destroyed[0] != "new-secret-id" {
		t.Fatalf("expected only the new secret_id to be destroyed, got %v", client.destroyed)
	}
}

func TestConditionallyRotateSecretId_CountsLookupErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "/lookup-error/secret-id", []byte("old-secret-id"), 0600); err != nil {
		t.Fatal(err)
	}
	client := &fakeApproleClient{lookupErr: errors.New("permission denied")}
	rotator, conf := newTestRotator(t, client, fs, "/lookup-error/secret-id")

	if err := rotator.ConditionallyRotateSecretId(conf, false); err == nil {
		t.Fatal("expected rotation to fail")
	}
	if got := testutil.ToFloat64(metrics.SecretIdRotationErrors.WithLabelValues(conf.SecretIdFile, "lookup")); got != 1 {
		t.Fatalf("expected one lookup error, got %v", got)
	}
	if client.generated != 0 {
		t.Fatalf("expected no secret_id to be generated, got %d", client.generated)
	}
}

func TestWriteSecretIdFile(t *testing.T) {
	tests := []struct {
		name string
		// existingMode is the mode of the existing secret_id file, 0 if there is none
		existingMode os.FileMode
		wantMode     os.FileMode
	}{
		{name: "new file", wantMode: 0600},
		{name: "keeps mode of existing file", existingMode: 0640, wantMode: 0640},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			secretIdFile := filepath.Join(dir, "secret-id")
			if tt.existingMode != 0 {
				if err := os.WriteFile(secretIdFile, []byte("old-secret-id"), tt.existingMode); err != nil {
					t.Fatal(err)
				}
				// not affected by the umask
				if err := os.Chmod(secretIdFile, tt.existingMode); err != nil {
					t.Fatal(err)
				}
			}
			rotator, conf := newTestRotator(t, &fakeApproleClient{}, afero.NewOsFs(), secretIdFile)

			if err := rotator.writeSecretIdFile("new-secret-id", conf); err != nil {
				t.Fatalf("writeSecretIdFile() unexpected error: %v", err)
			}

			data, err := os.ReadFile(secretIdFile)
			if err != nil || string(data) != "new-secret-id" {
				t.Fatalf("expected new secret_id, got %q (%v)", data, err)
			}
			info, err := os.Stat(secretIdFile)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != tt.wantMode {
				t.Fatalf("expected mode %v, got %v", tt.wantMode, info.Mode().Perm())
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected no temporary files to be left, got %v", entries)
			}
		})
	}
}
