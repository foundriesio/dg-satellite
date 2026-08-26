# About

This directory contains tools useful for local development

## `auth-config-*.json`

Authentication provider sample configurations for GitHub, Google, and local
username/password. See [configuring authentication](../docs/auth.md).

## `dev-shell` / `Dockerfile.devshell`

This script builds a container with all the required dependencies for
developing on this code base and will drop you in a container with the
project source code mounted.

## docker-compose.yml

This compose project builds and launches an update server that devices can
communicate with, storing its state in `.compose-server-data/` at the
repository root. See [Running in a Container](../docs/container.md) for
setup and usage.

## `e2e`

An end-to-end test suite that exercises a real update server together with
a real device client (`fioup`), driving update, config, and remote-action
flows through both the `fiocli` CLI and the web UI. See `e2e/README.md`.

## gen-certs.sh / fake-device.py

`gen-certs.sh` creates minimal fake data to stand up an update server and
have fake-devices connect to it.

`fake-device.py` is a simple script to issue HTTP requests against the server.

Example:
```
 $ ./contrib/gen-certs.sh /tmp/server
 $  go run github.com/foundriesio/update-server/cmd/server serve --datadir /tmp/server

 # From another terminal:
 ./contrib/fake-device.py -d /tmp/server/fake-devices/device-1 /device
 < HTTP 200
 {"Uuid":"04581446-DB43-43B1-BAC3-DBA7D1328AAC","PubKey":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcD
QgAE9M4irNPAcO+3N+UZdfqH6M86IhGg\nC1X2xPHpE1q1JkPnJUYnOtoLPrCVERAQqN/2gzeJG3nl7fqKHrbzNRixgA==\n-----END PUBLIC KEY-----\n","UpdateName":"","Deleted":false,"LastSeen":1754345526,"IsProd":false}
```

## `perf-test`

Self-contained Locust-based mTLS performance test that seeds and drives
thousands of fake devices against the update server. See
`perf-test/README.md`.

## `run-local.sh`

Stands up a local development server in one step, defaulting its datadir to
`./.local-data`. See [running locally](../docs/run-locally.md).

## Server

The Dockerfile that builds the update server container image, used by
`docker-compose.yml` and [Running in a Container](../docs/container.md).

## Terraform

Packer and Terraform recipes for a single-instance AWS deployment. See the
[production guide](../docs/production.md).
