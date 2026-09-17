# syntax=docker/dockerfile:1
#
# Host agent image: stages the static linuxaid-cli and linuxaid-install onto the node
# and runs the OpenVox agent in the host's namespaces via nsenter (see
# deploy/entrypoint.sh). Runs as the per-node Job that the operator creates. The
# node's cert is provided pre-signed (obmondo-clientcert), so there is nothing to enroll.

# The build stage runs on the build platform and cross-compiles for the target one, so a
# multi-arch build only emulates the final stage.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=spike
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-X main.Version=${VERSION} -s -w" -o /out/linuxaid-cli ./cmd/linuxaid-cli && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-X main.Version=${VERSION} -s -w" -o /out/linuxaid-install ./cmd/linuxaid-install

FROM alpine:3.24
# git + openssh-client: apply mode clones the puppet code in-container into the
# /opt/obmondo hostPath (the host itself is not required to have git).
RUN apk add --no-cache bash util-linux ca-certificates git openssh-client
COPY --from=build /out/linuxaid-cli /out/linuxaid-install /usr/local/bin/
COPY deploy/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
