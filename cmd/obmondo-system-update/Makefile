PATH := ../../../bin:$(PATH)
export NAME=obmondo-system-update
export VERSION=1.4.0
export DIST=../../../dist
export MAINTAINER=Ashish Jaiswal <ashish@obmondo.com>
export PREFIX=/opt/obmondo/bin/
SOURCES := obmondo-system-update

.PHONY: all dep build clean test format vet lint run

packages: deb deb_arm rpm

all: build

lint: ## Lint the files
	@golangci-lint run --issues-exit-code=1

format: ## Format the files
	@go fmt ./...

vet: ## Vet the files
	@go vet ./...

test: ## Run unittests
	@go test -v ./...

dep: ## Get the dependencies
	@go get -v ./...

clean: ## Remove previous build
	@go clean

run:
	@go run ./$(SOURECS)

$(SOURCES): dep
	CGO_ENABLED=0 GOARCH=amd64 go build -a -v -ldflags="-extldflags=-static" -o amd64/$(SOURCES) .
	CGO_ENABLED=0 GOARCH=arm64 go build -a -v -ldflags="-extldflags=-static" -o arm64/$(SOURCES) .
	chmod +x amd64/$(SOURCES) arm64/$(SOURCES)

deb: $(DIST)/$(NAME)_$(VERSION)_amd64.deb
$(DIST)/$(NAME)_$(VERSION)_amd64.deb: $(SOURCES)
	cd amd64 && TARGET=deb fpm-single-file $(SOURCES)

deb_arm: $(DIST)/$(NAME)_$(VERSION)_arm64.deb
$(DIST)/$(NAME)_$(VERSION)_arm64.deb: $(SOURCES)
	cd arm64 && ARCHITECTURE=arm64 TARGET=deb fpm-single-file $(SOURCES)

rpm: $(DIST)/$(NAME)-$(VERSION)-1.x86_64.rpm
$(DIST)/$(NAME)-$(VERSION)-1.x86_64.rpm: $(SOURCES)
	cd amd64 && TARGET=rpm fpm-single-file $(SOURCES)
