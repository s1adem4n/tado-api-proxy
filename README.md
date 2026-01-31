# tado API Proxy

A self-hosted proxy for the tado API that manages and rotates OAuth tokens automatically. Includes a web UI for account management and request statistics.

[Looking for the legacy version?](https://github.com/s1adem4n/tado-api-proxy/tree/legacy)

> [!WARNING]
> **Use at your own risk.** tado actively detects and bans accounts with high request volumes from third-party tools. While this proxy implements measures to reduce detection, account bans are still possible.

<table align="center">
  <tr>
    <th>Home</th>
    <th>Statistics</th>
  </tr>
  <tr>
    <td><img src="screenshots/home.png" alt="Home" width="350" /></td>
    <td><img src="screenshots/statistics.png" alt="Statistics" width="350" /></td>
  </tr>
</table>

## Features

- **Automatic token management** – Refreshes and rotates OAuth tokens seamlessly
- **Multi-account support** – Balance requests across multiple tado accounts
- **Official API authorization** – Route requests through the official tado API client for reduced ban risk
- **Web UI** – Manage accounts, view tokens, and monitor request statistics
- **Request logging** – Track API usage with detailed statistics

## Quick Start

### 1. Install and Run

**Docker (recommended):**

```sh
mkdir -p /path/to/data
sudo chown -R 1000:1000 /path/to/data

docker run -d \
  -p 8080:8080 \
  -v /path/to/data:/config \
  --restart unless-stopped \
  -e SUPERUSER_EMAIL=admin@example.com \
  -e SUPERUSER_PASSWORD=changeme \
  ghcr.io/s1adem4n/tado-api-proxy:latest
```

**Binary:**

Download from the [releases page](https://github.com/s1adem4n/tado-api-proxy/releases), then:

```sh
SUPERUSER_EMAIL=admin@example.com SUPERUSER_PASSWORD=changeme \
  ./tado-api-proxy serve --dir ./pb_data --http :8080
```

### 2. Add Your Account

> [!TIP]
> Use a secondary account instead of your main tado account. Create a new account, invite it to your home, and add it to the proxy. This can protect your primary account from potential bans (but it is not guaranteed). See the [Reducing Ban Risks](#reducing-ban-risk) section for more tips on avoiding bans!

1. Open http://localhost:8080 and log in with your superuser credentials
2. Add a tado account (email + password)
3. Tokens for the web and mobile clients are created automatically

### 3. Authorize the Official API (Highly Recommended)

> [!IMPORTANT]
> **This step significantly reduces your risk of being banned.** The official API client has a separate rate limit, which is approved by tado. The proxy prioritizes routing requests through the official API when available to reduce the risk of getting banned. Only one authorization per home is needed, as the limit is shared per home!

1. In the web UI, click **Start Authorization** in the "Authorize Official API" section
2. Complete the authorization flow in your browser
3. Make sure you're logged into the correct tado account when accepting

### 4. Start Making Requests

Replace `https://my.tado.com` with your proxy URL:

```sh
curl http://localhost:8080/api/v2/me
```

## API Usage

### Basic Request

The proxy automatically selects an available token:

```sh
curl http://localhost:8080/api/v2/me
```

### Target a Specific Account

Use the `X-Tado-Email` header to force a specific account:

```sh
curl -H "X-Tado-Email: account@example.com" http://localhost:8080/api/v2/me
```

### Request Statistics

Get request statistics:

```sh
curl http://localhost:8080/api/stats
```

Returns:

```json
{
  "today": 123,
  "last_hour": 45,
  "last_24_hours": 678
}
```

### Rate limit header

The proxy returns the `Ratelimit` and `Ratelimit-Policy` with the combined rate limit of all tokens you have added. It is in the same format as in the official tado API, e. g.:

```
Ratelimit: "perday";r=4999
Ratelimit-Policy: "perday";q=5000;w=86400
```

> `r` is the remaining requests, `q` is the total allowed requests, and `w` is the time window in seconds.

### API Documentation

OpenAPI docs are available at http://localhost:8080/docs

## Integrations

### Home Assistant

#### Using tado_hijack

[tado_hijack](https://github.com/banter240/tado_hijack) supports using the proxy natively by changing an option (see dev branch). It also implements some obfuscations to reduce the possibility of getting banned by tado. Please refer to the documentation for more details!

#### Using the official integration

The official tado integration in Home Assistant does not support changing the API url to a custom one, so you won't be able to route the requests through the proxy by changing an option. Some users have reported success with changing the source code of the extension though:

<details>
<summary><strong>Changing the base URL in the source code</strong></summary>

First locate the `PyTado` package files.
For Docker they are at `/usr/local/lib/python3.13/site-packages/PyTado/http.py`.
Change the row `MY_API = "http://my.tado.com/api/v2/"` to `MY_API = "http://localhost:8080/api/v2/"` (or your proxy URL). Restart Home Assistant and it should now use the proxy for API calls.

</details>

### Homebridge

The [homebridge-tado](https://github.com/homebridge-plugins/homebridge-tado) plugin supports custom API URLs. Point it to your proxy instance.

See these discussions for setup details:

- [Changing the base URL](https://github.com/homebridge-plugins/homebridge-tado/issues/176#issuecomment-3419839118)
- [Docker/Systemd setup](https://github.com/homebridge-plugins/homebridge-tado/issues/176#issuecomment-3421497695)

## Reducing Ban Risk

tado employs multiple detection methods from my research and testing:

| Method                  | Description                                                                |
| ----------------------- | -------------------------------------------------------------------------- |
| **IP-based limits**     | Seems to be about 5,000 requests per IP                                    |
| **Client-based limits** | Measured over a longer timefram (24 hours?); excessive usage triggers bans |
| **Pattern detection**   | Regular intervals (e.g., every 30s) appear suspicious                      |
| **Fingerprinting**      | Unusual client fingerprints result in account deletion                     |

Account treatment varies based on tado device ownership, account age, and other factors. My test accounts (temporary emails, no tado devices) were often deleted within 24–72 hours.

### Recommended Setup

For the most stable configuration:

1. **Authorize the Official API** – This is the single most effective step. The proxy routes through the official client first, which has a separate rate limit and is completely unbannable. Only one authorization per home is needed, as the limit is shared per home!

2. **Add multiple accounts** – Two accounts sharing your home seems to be the sweet spot. The proxy balances requests across their clients automatically.

3. **Use secondary accounts** – This can protect your main account to some degree.

<details>
<summary><strong>How to create extra accounts</strong></summary>

1. Open a private browser window
2. Use another legitimate email (recommended) or create a temporary email at [temp-mail.org](https://temp-mail.org)
3. Register at [login.tado.com](https://login.tado.com) — **don't** create a new home
4. From your main account, invite the new email to your home
5. Accept the invitation in the private window
6. Add the account to the proxy

</details>

### Tips for Developers

If you're building tools that use this proxy, please use these tips to decrease detection possibility:

- **Randomize request intervals** – Add jitter instead of fixed polling
- **Reduce overnight activity** – Lower request frequency during sleep hours
- **Batch requests** – Spread bursts over time instead of sending them all at once

## Configuration

The server uses [PocketBase](https://pocketbase.io). All PocketBase CLI flags work (`serve --dir`, `--http`, etc.).

| Environment Variable | Description                | Required     |
| -------------------- | -------------------------- | ------------ |
| `SUPERUSER_EMAIL`    | Initial superuser email    | On first run |
| `SUPERUSER_PASSWORD` | Initial superuser password | On first run |

## Building from Source

Requires Go 1.25+ and Bun.

```sh
git clone https://github.com/s1adem4n/tado-api-proxy.git
cd tado-api-proxy/web
bun install --frozen-lockfile && bun run build
cd ..
SUPERUSER_EMAIL=admin@example.com SUPERUSER_PASSWORD=changeme \
  go run cmd/main.go serve --dir ./pb_data --http :8080
```

## Credits

- [kritsel/tado-openapispec-v2](https://github.com/kritsel/tado-openapispec-v2) – Community OpenAPI specification
- [pocketbase/pocketbase](https://github.com/pocketbase/pocketbase) – Embedded database and admin API
- [scalar/scalar](https://github.com/scalar/scalar) – API documentation viewer
- [wmalgadey/PyTado](https://github.com/wmalgadey/PyTado) – Disclaimer inspiration

## Disclaimer

This software is provided for educational purposes only, "as is" without warranty of any kind, under the MIT license.

I am not affiliated with, associated with, or endorsed by tado° GmbH. This project was created for personal experimentation with IoT systems. Please use responsibly.
