FROM golang:1.26.4 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/taskflow ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/taskflow-migrate ./cmd/migrate

FROM alpine:3.22

RUN adduser -D -H -u 10001 appuser && apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/taskflow /usr/local/bin/taskflow
COPY --from=builder /out/taskflow-migrate /usr/local/bin/taskflow-migrate

USER appuser
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/taskflow"]
