# Data Model

## Devices

Devices will store simple attributes in sqlite and bigger payloads like
hardware-info directly on disk. The database model is:
```
CREATE TABLE devices (
				uuid VARCHAR(48) NOT NULL PRIMARY KEY,
				pubkey TEXT,
				deleted BOOL,
				is_prod BOOL,
				created_at INT,
				last_seen INT,
				tag VARCHAR(80),
				update_name VARCHAR(80),
				target_name VARCHAR(80),
				ostree_hash VARCHAR(80),
				apps JSON
)
```

All of these fields are set by the device with the exception of the
`deleted` and `update_name` attributes which will be manage by the
user-facing API.

Hardware-info, aktoml, network info, and update events can be stored
as files under a directory, `<DATADIR>/devices/<uuid>`.

## Updates

Updates represent copy of an offline update. They are organized in
hierarchy that mimics how the device API will need to look up content
to serve a device:
```
  <DATADIR>/updates/
                    CI | Production/
                                    <tag>/
                                          <update-name>/
                                                        Content (TUF, OSTree, containers)
```

For example, we might have two CI updates, for `main`, called `42` and
`43` as well as production update for `prod` called `40`. It would look like:
```
 <DATADIR>/updates
                  /ci/main/42/.....
                  /ci/main/43/.....
                  /prod/prod/40/.....
```

When a device asks for `targets.json` we would:
 * Look at its `isProd` x509 attribute
 * Look at its `x-ats-tags` header
 * Look up the device in our DB to find the update-name we should serve
 * Serve the content from that directory

## Rollouts - 50% thought out

Rollouts are how we configure a device to take a particular update. A
rollout consists of:

 * A specific update
 * A group of devices to take this update

We need to be able to create/list/update rollouts. We also need to
track the progress of a rollout so we can render in our UIs. I'm on
the fence about doing this in sqlite or files. I'm leaning to files.

Rollouts could be structured on disk like:
```
  <DATADIR>/rollouts/
                     <rollout-id>/
                                  rollout.json (created-at, isProd, tag, update-name)
                                  devices.json (list of devices)
                                  events - a file where device events are appended
                     active/ - a symlink to current rollout
```

The `events` file is a rough idea where can append events (file
append/write) can be done multi process/thread atomically in Linux.
The file could look like:
```
DEVICE_SEEN: <uuid>
DEVICE_STATUS: <uuid> DownloadStarted
DEVICE_STATUS: <uuid> DownloadComplete - true
```

The API semantics for a rollout could then be:

 * `POST rollout (isProd, tag, update-name) -> returns an ID`
 * `PUT rollout/<id>/devices` add device(s) to rollout
 * `PUT rollout/<id>/start` start the rollout (assign update-name attribute to devices in DB)
 * `GET rollout/<id>/tail` Tail with SSE events the progress of the rollout?
