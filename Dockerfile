# syntax=docker/dockerfile:1
#
# Host agent image: stages the static linuxaid-cli onto the node and runs the
# OpenVox agent in the host's namespaces via nsenter (see deploy/entrypoint.sh).
# Runs as the per-node Job that the fan-out launcher spawns. The node's cert is
# provided pre-signed (obmondo-clientcert), so there is nothing to enroll.

FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=spike
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-X main.Version=${VERSION} -s -w" -o /out/linuxaid-cli ./cmd/linuxaid-cli

FROM alpine:3.20
RUN apk add --no-cache bash util-linux ca-certificates
COPY --from=build /out/linuxaid-cli /usr/local/bin/linuxaid-cli
COPY deploy/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
