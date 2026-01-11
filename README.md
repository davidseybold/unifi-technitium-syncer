# unifi-technitium-syncer

Sync UniFi Network clients into a Technitium DNS Server zone.

This tool:

- Queries the UniFi Network API for the client list
- Sanitizes client names into DNS-safe labels
- Ensures `A` records exist in a target Technitium zone (and deletes stale ones)
- Stores the UniFi client MAC address in the Technitium record `comments`

## Requirements

- Go (per `go.mod`)
- A UniFi Network controller that supports the API-key based integration endpoint
- A Technitium DNS Server instance with an API token

## Configuration

Configuration is loaded via `viper`:

- A local file named `unifi-technitium-sync.env` (optional)
- Environment variables (recommended for production)

The config keys are:

- `UNIFI_API_URL`
- `UNIFI_API_KEY`
- `UNIFI_SITE_ID`
- `TECHNITIUM_API_URL`
- `TECHNITIUM_API_TOKEN`
- `SYNC_ZONE`

### Example `unifi-technitium-sync.env`

```env
UNIFI_API_URL=https://unifi.example.com
UNIFI_API_KEY=replace-me
UNIFI_SITE_ID=default

TECHNITIUM_API_URL=http://dns.example.com:5380
TECHNITIUM_API_TOKEN=replace-me

SYNC_ZONE=home.example.com
```

Notes:

- `UNIFI_API_URL` should be the base URL of your controller (no trailing slash is safest).
- `TECHNITIUM_API_URL` should be the base URL of Technitium’s web UI/API (often `http://<host>:5380`).
- `SYNC_ZONE` must be an existing zone in Technitium.

## Running

Run directly:

```bash
go run ./cmd
```

Build a binary:

```bash
go build -o unifi-technitium-syncer ./cmd
./unifi-technitium-syncer
```

## Behavior

- Only `A` records are managed.
- Records are created/updated with:
  - `ttl=3600`
  - `overwrite=true`
  - `ptr=true` and `createPtrZone=true` (PTR record creation is enabled)
- Deletions are performed for `A` records in the zone that are not present in the current UniFi client list.


