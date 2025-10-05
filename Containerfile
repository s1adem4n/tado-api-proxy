FROM docker.io/chromedp/headless-shell:latest

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

RUN useradd appuser
USER appuser

ARG TARGETARCH
COPY build/main-${TARGETARCH} /app

ENV CHROME_EXECUTABLE=/headless-shell/headless-shell
ENV TOKEN_PATH=/config/token.json
ENV COOKIES_PATH=/config/cookies.json
ENV LISTEN_ADDR=:8080

VOLUME /config
EXPOSE 8080

ENTRYPOINT ["/app"]