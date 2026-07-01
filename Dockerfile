# syntax=docker/dockerfile:1
#
# linuxaid-agents: a thin launcher image for running the LinuxAid OpenVox agent
# on a Kubernetes node. The container is only a delivery + scheduling shell — it
# stages the static linuxaid-cli binary onto the host and runs the agent inside
# the host's namespaces via nsenter (see deploy/entrypoint.sh). Intended to run
# as a per-node Job. The node's cert is provided pre-signed (obmondo-clientcert),
# so no enrollment/linuxaid-install is needed.

# ---- build the static binary ----
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=spike
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-X main.Version=${VERSION} -s -w" -o /out/linuxaid-cli ./cmd/linuxaid-cli

# ---- runtime: launcher that nsenters into the host ----
FROM alpine:3.20
RUN apk add --no-cache bash util-linux ca-certificates
COPY --from=build /out/linuxaid-cli /usr/local/bin/linuxaid-cli
COPY deploy/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
