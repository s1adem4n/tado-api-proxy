FROM docker.io/alpine:latest

RUN useradd appuser
USER appuser


ARG TARGETARCH
COPY build/main-${TARGETARCH} /app

VOLUME /config
EXPOSE 8080

ENTRYPOINT ["/app", "serve", "--dir", "/config", "--http", ":8080"]