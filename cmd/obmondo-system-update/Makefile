PATH := ../../bin:$(PATH)
export NAME=obmondo-system-update
export VERSION=1.3.2
export DIST=../../dist
export MAINTAINER=Ashish Jaiswal <ashish@obmondo.com>
export PREFIX=/opt/obmondo/bin/
SOURCES := obmondo-system-update

.PHONY: all dep build clean test format vet lint run

packages: deb rpm

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
	CGO_ENABLED=0 go build -a -v -ldflags="-extldflags=-static" -o $(SOURCES) .
	chmod +x $(SOURCES)

deb: $(DIST)/$(NAME)_$(VERSION)_amd64.deb
$(DIST)/$(NAME)_$(VERSION)_amd64.deb: $(SOURCES)
	TARGET=deb fpm-single-file $(SOURCES)

rpm: $(DIST)/$(NAME)-$(VERSION)-1.x86_64.rpm
$(DIST)/$(NAME)-$(VERSION)-1.x86_64.rpm: $(SOURCES)
	TARGET=rpm fpm-single-file $(SOURCES)
