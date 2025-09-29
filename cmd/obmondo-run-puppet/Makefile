PATH := ../../bin:$(PATH)
export NAME=obmondo-run-puppet
export VERSION=1.1.1
export DIST=../../dist
export MAINTAINER=Ashish Jaiswal <ashish@obmondo.com>
export PREFIX=/opt/obmondo/bin/
SOURCES := run_puppet

packages: deb rpm deb_arm64


test: ## Run unittests
	@go test -v ./...

build: $(SOURCES)

run_puppet:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -v -ldflags="-X main.Version=${VERSION} -extldflags=-static" -o $(SOURCES)_amd64 main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -v -ldflags="-X main.Version=${VERSION} -extldflags=-static" -o $(SOURCES)_arm64 main.go
	chmod +x $(SOURCES)_amd64
	chmod +x $(SOURCES)_arm64
	mv $(SOURCES)_amd64 $(SOURCES)

deb: $(SOURCES)
	ARCHITECTURE=amd64 TARGET=deb fpm-single-file $(SOURCES)

deb_arm64:
	mv $(SOURCES)_arm64 $(SOURCES)
	ARCHITECTURE=arm64 TARGET=deb fpm-single-file $(SOURCES)

rpm: $(DIST)/$(NAME)-$(VERSION)-1.x86_64.rpm
$(DIST)/$(NAME)-$(VERSION)-1.x86_64.rpm: $(SOURCES)
	TARGET=rpm fpm-single-file $(SOURCES)
