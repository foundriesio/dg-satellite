# Device Gateway Satellite Server

The satellite server is an open source system for managing Foundries.io™ update agents.
There are two primary use cases for the satellite server:

* Offline (air-gapped) environments where devices can not reach the Foundries.io backend
* Users who want to manage their own device management solution.

This project handles both use cases by implementing all the APIs used by Foundries.io update agents.
The project also includes a user-facing REST API and Web UI for managing devices and updates.

## Quick Start

Follow the [Quick Start](./docs/quick-start.md) guide to get a server running in development mode.

## Adding Updates

The satellite server uses content from [Offline Updates](https://docs.foundries.io/latest/user-guide/offline-update/offline-update.html)
to serve devices their TUF, OSTree, and Container data.
Follow the [updates](./docs/updates.md) guide for setting this up.

## API Access

Follow the [API](./docs/api.md) to learn how to access and use the REST API.

## Configuring Authentication Options

Follow the [configuring authentication](./docs/auth.md) guide for chosing the
method that meets your requirements.

## Running in Production

The [production guide](./docs/production.md) covers considerations when
deploying the satellite server for production use.

## Developing

The project is a single Golang binary that can be built with:

```
 go build github.com/foundriesio/dg-satellite/cmd/server
```

A "devshell" is also included that can be used for local development:

```
 ./contrib/dev-shell
```

***NOTE***: This repository uses Git-LFS. You'll need this installed to use the web UI.

## License

*dg-satellite* is under the [BSD 3-Clause Clear](./LICENSE.txt) license.
