# Updates

## Producing and Serving Update Content

The satellite server uses a specific file hierarchy for serving update
content:

```
 <datadir>/updates/
                   <ci or prod>/
                                <device tag>/
                                             <update>
```

The contents of the `update` directory above are the *exact* contents
of what `fioctl targets offline-update` produces. Let's say we want to
deploy a CI build to some devices connected to our satellite server.
The Target is tagged with `main` and is named `intel-corei7-64-lmp-148`.
The content can be staged on the satellite server by running:

```
  cd <datadir>
  mkdir -p updates/ci/main/148
  fioctl targets offline-update --expires-in-days 180 --tag main intel-corei7-64-lmp-148 ./updates/ci/main/148
```

This update will now show up under the "updates" view in your UI.

## Updating Your Devices

With an update in place, you'll need to create a "rollout" for your
device(s) to see it. This can be done from API, CLI, or Web UI.

### Updating via the API

Create a rollout named "first-try"

```
  curl \
    -H 'Authorization: Bearer <your token>' \
    -H 'Content-type: application/json' \
    -X PUT \
    -d '{"uuids": ["uuid1", "uuid2",...]}' \
    http://<your server>/v1/updates/ci/main/148/rollouts/first-try
```

### Updating via the CLI

Use the `satcli updates create-rollout` command.

### Updating via the Web

Scroll down to the specific update and click "Create rollout".

## Tracking the Progress of an Update/Rollout

You can track the progress of an update through the API, CLI, or Web.

### Tracking Progress With the API

There are two API resources for tailing updates. Both resources emit
[Server Sent Events](https://en.wikipedia.org/wiki/Server-sent_events).

* **rollout** — `/v1/updates/<ci|prod>/<tag>/<update>/rollouts/<rollout>/tail`
* **the whole update** — `/v1/updates/<ci|prod>/<tag>/<update>/tail`

### Tracking Progress With the CLI

The CLI has an `updates tail` subcommand that allows you to tail the update
or a specific rollout.

### Tracking Progress via the Web

Click "Follow progress" on either the Update or Rollout to see details.
