# tado API Proxy
A proxy server that bypasses tado's API rate limits by using OAuth2 authentication. Supports both browser-based and mobile app authentication methods.

> [!WARNING]  
> tado has started locking out users from using their official apps when using the proxy with a high request volume. Use at you own caution!

## Disclaimer
Besides owning a tado system, I have no connection with the tado company themselves. 
`tado-api-proxy` was created for my own use, and for others who may wish to experiment with personal Internet of Things systems. 
I have no business interest with tado. 
This software is provided without warranty, according to the MIT license.
This software was made for purely education purposes, and should not be used with bad intentions.

## Installation
### Docker
The container includes a headless Chromium browser (required only for browser auth method).

Create a data directory with proper permissions:
```sh
mkdir -p /path/to/data
sudo chown -R 1000:1000 /path/to/data
```

Run the container with mobile auth (recommended - simpler, no Chrome needed):
```sh
docker run -d \
  -p 8080:8080 \
  -v /path/to/data:/config \
  --restart unless-stopped \
  -e AUTH_METHOD=mobile \
  -e EMAIL=you@email.com \
  -e PASSWORD=yourpassword \
  ghcr.io/s1adem4n/tado-api-proxy:latest
```

Or run with browser auth (original method):
```sh
docker run -d \
  -p 8080:8080 \
  -v /path/to/data:/config \
  --restart unless-stopped \
  -e AUTH_METHOD=browser \
  -e EMAIL=you@email.com \
  -e PASSWORD=yourpassword \
  ghcr.io/s1adem4n/tado-api-proxy:latest
```

If you encounter file permission errors, ensure the data directory is writable by user `1000:1000`.


### Binary
Download the latest release from the [releases page](https://github.com/s1adem4n/tado-api-proxy/releases).

Set your credentials and run with mobile auth (recommended):
```sh
export AUTH_METHOD=mobile
export EMAIL=you@email.com
export PASSWORD=yourpassword
./tado-api-proxy
```

Or run with browser auth:
```sh
export AUTH_METHOD=browser
export EMAIL=you@email.com
export PASSWORD=yourpassword
export CHROME_EXECUTABLE=/usr/bin/chromium  # Only needed for browser auth
./tado-api-proxy
```


### From Source
Requires Go 1.25 or later.

```sh
git clone https://github.com/s1adem4n/tado-api-proxy.git
cd tado-api-proxy
export AUTH_METHOD=mobile
export EMAIL=you@email.com
export PASSWORD=yourpassword
go run cmd/main.go
```


## Usage
Replace `https://my.tado.com` with `http://localhost:8080` in your API calls.

For example, to get your profile:
```sh
curl http://localhost:8080/api/v2/me
```

API documentation is available at `http://localhost:8080/docs`.

### Statistics
The proxy tracks request volume to help you monitor usage. You can view these statistics at `http://localhost:8080/stats`.

Example response:
```json
{
  "today": 42,
  "last_hour": 5,
  "last_24_hours": 120
}
```


### Integration with Home Assistant
Currently, there is no simple way to change the API base URL in the official tado integration. However, you can edit the integration code to replace the base URL with your proxy's URL.

To do this, locate the `PyTado` package files. 
For Docker they are at `/usr/local/lib/python3.13/site-packages/PyTado/http.py`.
Change the row `MY_API = "http://my.tado.com/api/v2/"` to `MY_API = "http://localhost:8080/api/v2/"` (or your proxy URL). Restart Home Assistant and it should now use the proxy for API calls.

Another problem is that the `PyTado` library depends on the `device_code` OAuth2 flow, which is not supported by the proxy (as it automatically handles the authentication). Heavy modifications to the integration code may be required to bypass this, but users [have reported success](https://community.home-assistant.io/t/tado-rate-limiting-api-calls/928751/41) with only doing the above change.

Please see this [Home Assistant community thread](https://community.home-assistant.io/t/tado-rate-limiting-api-calls/928751) for more details.

PRs to add more info about Home Assistant integration are very welcome!


### Integration with Homebridge
The Homebridge plugin supports changing the API base URL. Just point it to your proxy instance and it should *just work*™.

For more details about setting it up, see these comments:
- [Changing the base URL](https://github.com/homebridge-plugins/homebridge-tado/issues/176#issuecomment-3419839118)
- [Details about running with Docker/Systemd](https://github.com/homebridge-plugins/homebridge-tado/issues/176#issuecomment-3421497695)


## Configuration
Currently, configuration is only possible via environment variables:
| Variable          | Description                                   | Default                                | Required |
| ----------------- | --------------------------------------------- | -------------------------------------- | -------- |
| EMAIL             | tado account email                            | *none*                                 | Yes      |
| PASSWORD          | tado account password                         | *none*                                 | Yes      |
| AUTH_METHOD       | Authentication method (`browser` or `mobile`) | `browser`                              | No       |
| LISTEN_ADDR       | Server listen address                         | `:8080`                                | No       |
| TOKEN_PATH        | Token storage file                            | `token.json`                           | No       |
| COOKIES_PATH      | Cookies storage file (browser auth only)      | `cookies.json`                         | No       |
| CHROME_EXECUTABLE | Chrome/Chromium path (browser auth only)      | `/usr/bin/chromium`                    | No       |
| BROWSER_TIMEOUT   | Browser auth timeout (browser auth only)      | `5m` (5 minutes)                       | No       |
| HEADLESS          | Run browser headless (browser auth only)      | `true`                                 | No       |
| TIMEZONE          | Timezone (mobile auth only)                   | `Europe/Berlin`                        | No       |
| CLIENT_ID         | OAuth2 client ID                              | `af44f89e-ae86-4ebe-905f-6bf759cf6473` | No       |
| LOG_LEVEL         | Log level (`debug`, `info`, `warn`, `error`)  | `info`                                 | No       |


## How It Works

### Mobile Auth Flow
The proxy uses OAuth2 with PKCE to authenticate:
1. Generates a PKCE code challenge and verifier
2. Sends credentials to Tado's OAuth authorize endpoint
3. Follows the redirect chain to obtain an authorization code
4. Exchanges the code for access and refresh tokens
5. Uses the access token to authenticate API requests
6. Automatically refreshes tokens when they expire

### Browser Auth Flow
The proxy launches a headless Chrome instance to authenticate with tado using your credentials. It extracts the OAuth2 token from browser storage and uses it to authenticate API requests. When tokens expire, the refresh token is used to obtain new ones. After 2-3 days when refresh tokens expire, the browser authentication process repeats automatically.


## Credits
- [kritsel/tado-openapispec-v2](https://github.com/kritsel/tado-openapispec-v2) - Community OpenAPI specification
- [go-rod/rod](https://github.com/go-rod/rod) - Browser automation library
- [scalar/scalar](https://github.com/scalar/scalar) - API documentation viewer
- [wmalgadey/PyTado](https://github.com/wmalgadey/PyTado) - Disclaimer inspiration :P
