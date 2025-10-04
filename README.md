# Tado API Proxy
A proxy server to bypass Tado's new rate limits on their public API. It uses their browser OAuth2 client to authenticate requests, which has much higher rate limits.

## Usage
Run the container with your credentials:
```sh
docker run -d \
  -p 8080:8080 \
  -v /path/to/config:/config \
  --restart unless-stopped \
  -e EMAIL=you@email.com \
  -e PASSWORD=yourpassword \
  ghcr.io/s1adem4n/tado-api-proxy:latest
```

Then access the API like you would normally do, except replacing `https://my.tado.com` with `http://localhost:8080`.

For example, to get your profile:
```sh
curl http://localhost:8080/api/v2/me
```

## Configuration
| Environment Variable | Description                        | Default             |
| -------------------- | ---------------------------------- | ------------------- |
| LISTEN_ADDR          | Address to listen on               | `:8080`             |
| TOKEN_PATH           | Path to token file                 | `token.json`        |
| COOKIES_PATH         | Path to cookies file               | `cookies.json`      |
| EMAIL                | Tado email address                 | *required*          |
| PASSWORD             | Tado password                      | *required*          |
| CHROME_EXECUTABLE    | Path to Chrome/Chromium executable | `/usr/bin/chromium` |


## How it works
The proxy server uses a headless Chromium browser to log in, and then extracts the OAuth2 token from the browser's local storage.
It then uses this token to authenticate all requests to Tado's API.
The token is automatically refreshed using a standard OAuth2 refresh token flow, without needing to run the browser again.
However, the token can be refreshed only a limited number of times, and after about 2-3 days, the browser needs to be run again to get a new token. 
This is done automatically by the proxy server when it detects that the token has expired and cannot be refreshed anymore.