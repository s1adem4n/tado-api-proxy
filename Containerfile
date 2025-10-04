FROM docker.io/golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /tado-api-proxy ./cmd/tado-api-proxy

FROM docker.io/alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=build /tado-api-proxy .

ENV TOKEN_PATH=/config/token.json
ENV COOKIES_PATH=/config/cookies.json
VOLUME /config
ENV LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/tado-api-proxy"]