FROM docker.io/alpine:latest

RUN apk --no-cache add ca-certificates chromium

ARG TARGETARCH
COPY build/main-${TARGETARCH} /app

ENV CHROME_EXECUTABLE=/usr/bin/chromium
ENV TOKEN_PATH=/config/token.json
ENV COOKIES_PATH=/config/cookies.json
ENV LISTEN_ADDR=:8080

VOLUME /config
EXPOSE 8080

ENTRYPOINT ["/app"]