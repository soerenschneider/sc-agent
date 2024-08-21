package vault_common

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/vault"
	"go.uber.org/multierr"
)

type VaultClient interface {
	List(path string) (*api.Secret, error)
	Read(path string) (*api.Secret, error)
	Write(path string, data map[string]any) (*api.Secret, error)
}

type CidrResolver interface {
	GetCidr() ([]string, error)
}

type ApproleSecretIdRotatorClient struct {
	client            VaultClient
	mountPath         string
	tokenCidrResolver CidrResolver
	cidrListResolver  CidrResolver
	ttl               string
}

type ApproleSecretIdRotationOpts func(client *ApproleSecretIdRotatorClient) error

func NewClient(client VaultClient, mountPath string, opts ...ApproleSecretIdRotationOpts) (*ApproleSecretIdRotatorClient, error) {
	if client == nil {
		return nil, errors.New("empty client passed")
	}
	ret := &ApproleSecretIdRotatorClient{
		client:    client,
		mountPath: mountPath,
	}

	var errs error
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return ret, errs
}

func (a *ApproleSecretIdRotatorClient) getCidrList() ([]string, error) {
	if a.cidrListResolver != nil {
		return a.cidrListResolver.GetCidr()
	}

	return nil, nil
}

func (a *ApproleSecretIdRotatorClient) getTokenCidrList() ([]string, error) {
	if a.tokenCidrResolver != nil {
		return a.tokenCidrResolver.GetCidr()
	}

	return nil, nil
}

func (a *ApproleSecretIdRotatorClient) DestroySecretId(roleName, secretId string, isAccessor bool) error {
	pathName := "secret-id"
	if isAccessor {
		pathName = "secret-id-accessor"
	}

	reqData := map[string]any{
		strings.ReplaceAll(pathName, "-", "_"): secretId,
	}

	path := fmt.Sprintf("auth/%s/role/%s/%s/destroy", a.mountPath, roleName, pathName)
	_, err := a.client.Write(path, reqData)
	return err
}

func (a *ApproleSecretIdRotatorClient) Lookup(roleName, secretId string, isAccessor bool) (*vault.SecretIdInfo, error) {
	pathName := "secret-id"
	if isAccessor {
		pathName = "secret-id-accessor"
	}
	reqData := map[string]any{
		strings.ReplaceAll(pathName, "-", "_"): secretId,
	}

	path := fmt.Sprintf("auth/%s/role/%s/%s/lookup", a.mountPath, roleName, pathName)
	resp, err := a.client.Write(path, reqData)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, errAlreadyExpired
	}

	secretIdInfo, err := vault.ParseLifetimeIdInfo(resp.Data)
	if err != nil {
		return nil, err
	}

	return &secretIdInfo, nil
}

func (a *ApproleSecretIdRotatorClient) GetSecretIdAccessors(roleName string) ([]string, error) {
	path := fmt.Sprintf("/auth/approle/role/%s/secret-id", roleName)
	resp, err := a.client.List(path)
	if err != nil {
		return nil, err
	}

	keysRaw, found := resp.Data["keys"]
	if !found {
		return nil, errors.New("no keys found")
	}

	interfaceSlice, ok := keysRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value associated with key %s is not a slice", keysRaw)
	}

	stringSlice := make([]string, len(interfaceSlice))
	for i, v := range interfaceSlice {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("element %d in the slice is not a string", i)
		}
		stringSlice[i] = str
	}

	return stringSlice, nil
}

func (a *ApproleSecretIdRotatorClient) GenerateSecretId(roleName string) (string, error) {
	path := fmt.Sprintf("auth/%s/role/%s/secret-id", a.mountPath, roleName)
	resp, err := a.client.Write(path, a.generateSecretIdRotationData())
	if err != nil {
		return "", fmt.Errorf("unable to create new secret_id: %w", err)
	}

	respData, found := resp.Data["secret_id"]
	if !found {
		return "", errors.New("no field 'secret_id' in response")
	}

	converted, conversionOk := respData.(string)
	if !conversionOk {
		return "", errors.New("could not convert 'secret_id' to string")
	}

	return converted, nil
}

func (a *ApproleSecretIdRotatorClient) ReadRoleId(roleName string) (string, error) {
	path := fmt.Sprintf("auth/%s/role/%s/role-id", a.mountPath, roleName)
	secret, err := a.client.Read(path)
	if err != nil {
		return "", err
	}

	data, found := secret.Data["role_id"]
	if !found {
		return "", errors.New("no field 'role_id' in response")
	}

	converted, conversionOk := data.(string)
	if !conversionOk {
		return "", errors.New("could not convert 'role_id' to string")
	}

	return converted, nil
}

func getMetadata() string {
	metadata := map[string]string{
		"managed-by": "sc-agent",
	}

	marshalled, err := json.Marshal(metadata)
	if err != nil {
		log.Error().Err(err).Str(logComponent, approleComponentName).Msg("could not marshall metadata")
	}

	return string(marshalled)
}

func (a *ApproleSecretIdRotatorClient) generateSecretIdRotationData() map[string]any {
	data := map[string]any{
		"metadata": getMetadata(),
	}

	cidrList, err := a.getCidrList()
	if err != nil {
		log.Error().Err(err).Str(logComponent, approleComponentName).Msgf("could not generate cidr list")
	} else {
		data["cidr_list"] = cidrList
	}

	tokenCidrList, err := a.getTokenCidrList()
	if err != nil {
		log.Error().Err(err).Str(logComponent, approleComponentName).Msgf("could not generate token cidr list")
	} else {
		data["token_bound_cidrs"] = tokenCidrList
	}

	if a.ttl != "" {
		data["ttl"] = a.ttl
	}

	return data
}
