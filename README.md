# tado API Proxy
A self-hosted proxy for the tado API that stores and rotates OAuth tokens on your server. It includes a built-in web UI for managing accounts and tokens.

> [!WARNING]
> tado has started locking out users from using their official apps when using the proxy with a high request volume. Use at your own caution!

## Disclaimer
Besides owning a tado system, I have no connection with the tado company themselves.
`tado-api-proxy` was created for my own use, and for others who may wish to experiment with personal Internet of Things systems.
I have no business interest with tado.
This software is provided without warranty, according to the MIT license.
This software was made for purely education purposes, and should not be used with bad intentions.

## Features
- Web admin UI for managing accounts, tokens, and device-code authorizations
- Multiple tado accounts with different homes supported
- Balances usage between tokens
- Automatic token refresh and request logging
- Optional authentication to the official tado API client

## Installation
### Docker
Create a data directory with proper permissions:
```sh
mkdir -p /path/to/data
sudo chown -R 1000:1000 /path/to/data
```

Run the container and create the initial superuser:
```sh
docker run -d \
  -p 8080:8080 \
  -v /path/to/data:/config \
  --restart unless-stopped \
  -e SUPERUSER_EMAIL=admin@example.com \
  -e SUPERUSER_PASSWORD=change-me \
  ghcr.io/s1adem4n/tado-api-proxy:latest
```

### Binary
Download the latest release from the [releases page](https://github.com/s1adem4n/tado-api-proxy/releases).

```sh
export SUPERUSER_EMAIL=admin@example.com
export SUPERUSER_PASSWORD=change-me
./tado-api-proxy serve --dir ./pb_data --http :8080
```

### From Source
Requires Go 1.25.5+ and Bun.

```sh
git clone https://github.com/s1adem4n/tado-api-proxy.git

cd tado-api-proxy/web
bun install --frozen-lockfile
bun run build

cd ..
export SUPERUSER_EMAIL=admin@example.com
export SUPERUSER_PASSWORD=change-me
go run cmd/main.go serve --dir ./pb_data --http :8080
```

## Usage
1. Open the admin UI at http://localhost:8080 and log in with your superuser account.
2. Add one or more tado accounts (email + password). Tokens for the web and mobile apps are created automatically.
3. (Optional) Use the “Authorize Official API” section to complete the device-code flow for the official API client.
4. Send API requests to the proxy by replacing https://my.tado.com with your proxy base URL.

Example:
```sh
curl http://localhost:8080/api/v2/me
```

If you have multiple accounts, you can select one with the `X-Tado-Email` header. If you don't specify one, the proxy will balance it across all the accounts.
```sh
curl -H "X-Tado-Email: you@email.com" http://localhost:8080/api/v2/me
```

OpenAPI documentation is available at http://localhost:8080/docs.

## Configuration
The server is a PocketBase app. Runtime flags are the same as PocketBase (`serve --dir`, `--http`, etc.).

Environment variables:
| Variable           | Description                                    | Required |
| ------------------ | ---------------------------------------------- | -------- |
| SUPERUSER_EMAIL    | Initial superuser email (created on first run) | Yes*     |
| SUPERUSER_PASSWORD | Initial superuser password                     | Yes*     |

*Only required if no superuser exists yet.


## Credits
- [kritsel/tado-openapispec-v2](https://github.com/kritsel/tado-openapispec-v2) - Community OpenAPI specification
- [pocketbase/pocketbase](https://github.com/pocketbase/pocketbase) - Embedded database and admin API
- [scalar/scalar](https://github.com/scalar/scalar) - API documentation viewer
- [wmalgadey/PyTado](https://github.com/wmalgadey/PyTado) - Disclaimer inspiration :P
