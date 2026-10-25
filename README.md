# sc-agent

[![Go Report Card](https://goreportcard.com/badge/github.com/soerenschneider/sc-agent)](https://goreportcard.com/report/github.com/soerenschneider/sc-agent)
![test-workflow](https://github.com/soerenschneider/sc-agent/actions/workflows/test.yaml/badge.svg)
![release-workflow](https://github.com/soerenschneider/sc-agent/actions/workflows/release-container.yaml/badge.svg)
![golangci-lint-workflow](https://github.com/soerenschneider/sc-agent/actions/workflows/golangci-lint.yaml/badge.svg)
![openapi-spec](https://github.com/soerenschneider/sc-agent/actions/workflows/openapi.yaml/badge.svg)


> Code, API and README is still work in progress

## Table of Contents

1. [Overview](#overview)
1. [Features](#features)
1. [Installation](#installation)
   1. [Configuration](#configuration)
1. [Development Workflow](#development-workflow)
1. [Security Considerations](#security-considerations)
1. [Components](#components)
    - [k0s](#k0s)
    - [libvirt](#libvirt)
    - [nftables](#nftables)
    - [packages](#packages)
    - [pki](#pki)
    - [secrets](#secrets)
    - [services](#services)
    - [system](#system)
    - [wake-on-lan](#wake-on-lan)

## Overview

Configurable daemon that provides common features needed on virtual machine instances running in my [hybrid cloud](https://github.com/soerenschneider/soeren.cloud).


## Features

🔑 Sync secrets from Vault<br/>
🏭 Manage x509 and SSH certificates<br/>
📦 Start, stop and restart libvirt domains and systemd units<br/>
📫 Monitor system updates<br/>
🚦 Automatic shutdown, reboot and waking-up of hardware unit<br/>

## Development Workflow

### OpenAPI Spec
Development is done "API first", therefore [server code](internal/adapters/http/api.gen.go) and [client code](pkg/api/api.gen.go) are auto-generated using [oapicodegen](https://github.com/oapi-codegen/oapi-codegen) and should not be changed by hand.

The OpenAPI 3 spec file is defined [here](openapi.yaml) and a *swagger* like page is available at the path `/docs` of the API server.

### Generating server and client code

```bash
> make generate
```

### Linting
Linting is done via spectral using the default OpenAPI configuration.

### Detecting API / code configuration drift
A GitHub Actions workflow is in-place that runs code-generation on every commit and fails if the generated code doesn't match the committed code.

## Security Considerations

### Verifying releases
Release binaries are built reproducibly (`make release`). The checksums are signed keylessly with
[cosign](https://github.com/sigstore/cosign) by the release workflow:

```shell
cosign verify-blob \
  --bundle checksum.sha256.sigstore.json \
  --certificate-identity "https://github.com/soerenschneider/sc-agent/.github/workflows/release-container.yaml@refs/heads/main" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksum.sha256
sha256sum -c --ignore-missing checksum.sha256
```

### Authentication
The default mode is using mTLS to authenticate against the server that serves the REST API. A middleware is available that validates request based on the `CommonName` attribute or `EmailAddresses` attributes of the certificate.

Although strongly discouraged, the server can be configured to run without any authentication for development purposes.

### Authorization
All successfully authenticated users share the same permissions, no distinguished roles are available.

### Vault AppRole bootstrapping
If the `secret_id_file` is missing, contains a wrapping token that can not be unwrapped anymore or Vault rejects the
`secret_id`, a new response-wrapped `secret_id` can be fetched from [approle-courier](https://github.com/soerenschneider/approle-courier).
The agent authenticates using an age-encrypted challenge, so its age identity must be registered with approle-courier.
Before unwrapping, the wrapping token's `creation_path` is verified. Attempts are rate limited with an exponential
backoff (1m up to 1h). Transient failures before a `secret_id` could have been created, e.g. while approle-courier is
unreachable, are retried sooner (1m up to 5m). Afterwards the `secret_id` is rotated as usual.

```yaml
vault:
  default:
    auth_method: approle
    role_id: 3c5e8a0b-1f2d-4e6a-9b7c-8d0e1f2a3b4c
    role_name: demo         # optional, defaults to role_id
    approle_mount: approle
    secret_id_file: /etc/sc-agent/secret-id
    bootstrap:
      address: https://approle-courier.example.com
      age_identity_file: /etc/sc-agent/age.key
      machine_id: node-42   # optional, defaults to the hostname
      role_name: demo       # optional, defaults to the client's role_name
      wrap_ttl: 2m          # optional
```

### Managing secret_ids of other AppRoles
Additional `vault` clients can manage the `secret_id`s of other AppRoles on the same Vault instance, e.g. for services
running on the same machine. Each AppRole rotates its own `secret_id`, so no AppRole needs permissions for other
AppRoles. Clients with `rotation_only: true` don't keep a token: they log in for each check and revoke the token
afterwards. Checks are scheduled based on the `secret_id`'s remaining lifetime (10m up to 24h), so logins are rare.
Rotation-only clients can not be used by other components and need the `default` policy (or permission for
`auth/token/revoke-self`). A short `token_ttl` for these AppRoles is recommended.

```yaml
vault:
  default:
    role_id: sc-agent
    secret_id_file: /etc/sc-agent/secret-id
  restic:
    role_id: 7a1c2e3f-4b5d-6e7f-8a9b-0c1d2e3f4a5b
    role_name: restic
    secret_id_file: /etc/restic/vault-secret-id
    secret_id_file_user: restic
    rotation_only: true
```

As checks can be up to 24h apart, alert on the `secret_id`'s expiration instead of its remaining lifetime in percent,
which is only updated by successful checks:

```
sc_agent_vault_approle_secret_id_expiration_timestamp_seconds - time() < 3 * 86400
```


## Components

### K0s
- start K0s service
- stop K0s service

### libvirt
- start a libvirt domain
- shutdown a libvirt domain
- restart a libvirt domain

### nftables
- add an IP address or prefix to a predefined named set with a timeout. If the element is already part of the set,
  its timeout is reset.

Only sets that are configured can be modified, they are addressed via their alias. The set must be created with the
`timeout` flag, sets containing prefixes additionally need the `interval` flag.

```yaml
nftables:
  enabled: true
  use_sudo: true
  sets:
    ssh-allowlist:          # alias used in the API
      family: inet
      table: filter
      set: ssh_allowed
      max_timeout: 24h      # optional
```

```bash
curl -X POST https://localhost:9999/v1/nftables/sets/ssh-allowlist/elements \
  -H 'Content-Type: application/json' -d '{"element": "192.0.2.10", "timeout": "1h"}'
```

Via MQTT, publish to `sc-agent/<topic_host_prefix>/nftables/add` (or a group topic) with the alias in the payload. The
result is published to `sc-agent/<topic_host_prefix>/nftables/response/add`.

```json
{"set": "ssh-allowlist", "element": "192.0.2.10", "timeout": "1h"}
```

### packages
- list installed packages on a system
- list updateable packages on a system
- upgrade all packages

### pki
- sign ssh public keys
- get ssh signatures configuration

### secrets
- replicate secrets from Hashicorp Vault to the local system
- get repliaction configuration

### services
- set status of system services (restarted, started, stopped)
- get logs of a system services

### system
- set power status of system (reboot, shutdown)
- get status of conditional-reboot
- set status of conditional-reboot (paused, unpaused)

### Wake-on-Lan

- Send WOL packets to wake up local machines
