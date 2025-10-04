FROM docker.io/golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /tado-api-proxy ./cmd/main.go

FROM docker.io/alpine:latest

RUN apk --no-cache add ca-certificates chromium

WORKDIR /app
COPY --from=build /tado-api-proxy .

ENV CHROME_EXECUTABLE=/usr/bin/chromium
ENV TOKEN_PATH=/config/token.json
ENV COOKIES_PATH=/config/cookies.json
ENV LISTEN_ADDR=:8080

VOLUME /config
EXPOSE 8080

ENTRYPOINT ["/app/tado-api-proxy"]