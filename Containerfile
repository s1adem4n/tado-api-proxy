FROM docker.io/alpine:latest

RUN apk add --no-cache tzdata && \
    addgroup -g 1000 -S appuser && adduser -u 1000 -S -G appuser appuser && \
    mkdir /config && chown appuser:appuser /config
USER appuser

ARG TARGETARCH
COPY build/main-${TARGETARCH} /app

VOLUME /config
EXPOSE 8080

ENTRYPOINT ["/app", "serve", "--dir", "/config", "--http", ":8080"]