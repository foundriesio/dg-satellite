# RFC — Gateway Mode

> [!WARNING]
> This document discusses functionality which is still in the process of being implemented.

There are two modes the satellite server can run in:

* Fully detached standalone — the server is running in an air-gapped facility.
* Gateway — the server is connected to the Internet. The devices are not.

In gateway mode, the satellite server functions as a bridge to
FoundriesFactory®. This document describes how this works.

There are two parts to the gateway mode:

* Sending updates from satellite server to FoundriesFactory with information on
  devices such as `last-seen`, `hwinfo`, and `update-events`.

* Receiving information from its Factory such as device config changes
  and OTAs that need to be performed.

## Security

The satellite server and device-gateway make use of an attribute to distinguish
production devices from regular devices:

```
  businessCategoryOid        = asn1.ObjectIdentifier{2, 5, 4, 15}
```

When set to `production`, we can cryptographically assert the Fleet operator's
intentions. This attribute could be used to denote a new type of device/mode
called `dg-satellite`. This would allow operators to have complete control over
what "devices" can function in gateway mode.

The gateway can then use this client cert when talking to the Foundries backend
to send and receive information.

## Foundries Backend

This new function can be amended to the device-gateway to piggy back on its
robust code base and test infrastructure. However, this new functionality should
run as a new standalone service that we can scale independently of the
device-gateway.

## Sending Changes to FoundriesFactory

Queue up changes happening on the satellite server and
publish them back to the Factory if *(think of this as "eventually
consistent")*:

* The number of changes crosses some size threshold
* A certain amount of time has elapsed (size 5 minutes)

We can add the notion of a "change listener" interface to the `storage/dg`
module so that a change listener is set on the `Storage` struct
handling incoming changes. The `dg_storage.go` code then calls the change
listener after committing changes locally.

The "gateway sync" change listener must work in a non-blocking manner. It must
also be robust enough to handle internet disconnects. E.g., saving these batches
of changes to disk is necessary in case someone tries to reboot the
satellite server hoping to fix a connection issue.

### Data Model

The data sent to the server is a tarball consisting of:

* `events.log` - Each line of the file is the details of a change event in
  in JSON. For example `{"event": "check-in", "details": {"time": 123, "device": "uuid"}}`
* attachments — Some events like recording hwinfo do not fit well in json. They can be
  bundled as attachments. For example:

   ```
   {"event": "put-file", "details": {"name": "aktoml", "device": "<uuid>", "attachment": "<uuid>_aktoml"}}
   ```

   As well as a file in the tarball named `<uuid>_aktoml`.
* tags — A list of tags (production and ci) that are being followed by devices
  attached to this server.

This payload can be periodically generated and sent to a backend service like:

```
   PUT https://<repoid>.satellite.foundries.io:8443/updates
```

## Receiving Changes to FoundriesFactory

There are two ways to do this. We are not excited about either:

* This new backend service can periodically scan through Couchbase for:
  * config changes
  * waves/rollouts (TBD)
* ota-lite could do a pub/sub function where changes are published
  to a queue. The new backend service listens to this queue.

An implication of this design is that device-groups must include a new
attribute to configure which satellite server (if any) is allowed to manage
configuration data.

The backend service then delivers this payload to a satellite server when it
requests it. e.g. `GET https://<repoid>.satellite.foundries.io:8443/updates`.

This backend service will be able to authenticate to our OSTree and Docker
registry with its client certificate.

### Receiving Data Model

The response back from the server will be a tarball consisting of:

* `config.json` - this a combination of:
  * fleet wide config
  * device group config
  * device specific configs
* `tuf.json` - TUF metadata for all tags this server handles.
* TBD — rollouts/waves/
