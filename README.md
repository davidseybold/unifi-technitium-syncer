# unifi-dns-sync

Sync UniFi Network clients into a DNS zone using a pluggable DNS provider.

This tool:

- Queries the UniFi Network API for the client list
- Sanitizes client names into DNS-safe labels
- Ensures `A` records exist in a target zone (and deletes stale ones)

## DNS providers

The syncer is designed to support multiple DNS backends via a provider interface.

Currently implemented:

- `technitium`

Future providers can be added under `dnsprovider/<name>` and wired up in `cmd/main.go`.

## Requirements

- Go (per `go.mod`)
- A UniFi Network controller that supports the API-key based integration endpoint
- A supported DNS provider (Technitium is supported today)

## Configuration

Configuration is loaded from:

- A local `.env` file (optional, loaded via `godotenv`)
- Environment variables (recommended for production)

The config keys are:

- `UNIFI_API_URL`
- `UNIFI_API_KEY`
- `UNIFI_SITE_ID`
- `DNS_PROVIDER` (e.g. `technitium`)
- `SYNC_ZONE`
- `STATE_DIR` (optional, defaults to `/var/lib/unifi-sync`)

### Provider-specific configuration

#### Technitium (`DNS_PROVIDER=technitium`)

- `TECHNITIUM_API_URL`
- `TECHNITIUM_API_TOKEN`

### Example `.env`

```env
UNIFI_API_URL=https://unifi.example.com
UNIFI_API_KEY=replace-me
UNIFI_SITE_ID=default

DNS_PROVIDER=technitium
TECHNITIUM_API_URL=http://dns.example.com:5380
TECHNITIUM_API_TOKEN=replace-me

SYNC_ZONE=home.example.com
STATE_DIR=/var/lib/unifi-sync
```

Notes:

- `UNIFI_API_URL` should be the base URL of your controller (no trailing slash is safest).
- `TECHNITIUM_API_URL` should be the base URL of Technitium’s web UI/API (often `http://<host>:5380`).
- `SYNC_ZONE` must be an existing zone in Technitium.
- `STATE_DIR` stores `state.json` used to remember recently-seen clients.

## Running

Run directly:

```bash
go run ./cmd
```

Build a binary:

```bash
go build -o unifi-dns-sync ./cmd
./unifi-dns-sync
```
