# Quick Start

## Prerequisites

Devices authenticate using mTLS and your FoundriesFactory® PKI. You'll
need access to your [Factory CA](https://docs.foundries.io/latest/reference-manual/security/device-gateway.html)
in order to create a TLS certificate for device-facing APIs.

## Building

First build the service with:

```
 go build -o dg-sat github.com/foundriesio/dg-satellite/cmd
```

## Configure Mutual TLS

### Create Certificate Signing Requests for TLS

Devices need to trust the TLS connection they make to this server. In
order to do this, you must create a CSR to be signed with the Factory
root key:

```
  ./dg-sat --datadir=./datadir create-csr --dnsname <HOSTNAME> --factory <FACTORY>
```

### Sign the Request

Copy `data/certs/tls.csr` to the computer with your factory PKI. This
file does not contain sensitive information, so it is safe to share as
needed. From the factory PKI directory run:

```
  <pkidir>/sign_tls_csr <path to tls.csr>
```

This script will print the contents of the certificate. The contents are
not sensitive. Go back to the satellite server system and create the
file `datadir/certs/tls.pem` with this content.

### Grant Access to Devices

This service needs to know what devices can connect to it. You can allow
all valid Factory devices to connect with:

```
 fioctl keys ca show --just-device-cas > datadir/certs/cas.pem
```

## Configure User Authentication

The satellite server includes a few [authentication providers](../auth)
for user-facing APIs. The "noauth" provider is handy for starting up a
quick local environment for testing and evaluation. Running
`auth-init --test` command will setup an HMAC encryption key for API
tokens and web sessions as well as the "noauth" provider.

```
  ./dg-sat --datadir=./datadir auth-init --test
```

## Run

`./dg-sat serve --datadir=datadir`

You can browse the UI at <http://localhost:8080/>

Devices can now connect to the server.
The `/var/sota/sota.toml` file has several "server" settings
that need to point at this new server:

* tls.server
* provision.server
* uptane.repo_server
* pacman.ostree_server
