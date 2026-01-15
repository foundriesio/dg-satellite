# Device Gateway Satellite Server
The satellite server is an open source system for managing Foundries.io update agents.
There are two primary use cases for the satellite server:

 * Offline (air-gapped) environments where devices can't reach the Foundries.io backend
 * Users who want to manage their own device management solution

This project handles both use cases by implementing all the APIs used by Foundries.io update agents.
The project also includes a user-facing REST API and Web UI for managing devices and updates.

## Quick Start
Follow the [Quick Start](./docs/quick-start.md) guide to get a server running in development mode.

## API access
Follow the [API](./docs/api.md) to learn how to access and use the REST API.

## Configuring authentication options
Follow the [configuring authentication](./docs/auth.md) guide for chosing the
method that meets your requirements.

## Developing
The project is a single Golang binary that can be built with:
```
 go build github.com/foundriesio/dg-satellite/cmd/
```

A "devshell" is also included that can be used for local development:
```
 ./contrib/dev-shell
```

## License
*dg-satellite* is under the [BSD 3-Clause Clear](./LICENSE.txt) license.
