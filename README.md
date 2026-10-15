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

### Authentication
The default mode is using mTLS to authenticate against the server that serves the REST API. A middleware is available that validates request based on the `CommonName` attribute or `EmailAddresses` attributes of the certificate.

Although strongly discouraged, the server can be configured to run without any authentication for development purposes.

### Authorization
All successfully authenticated users share the same permissions, no distinguished roles are available.


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
