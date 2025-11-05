PATH := ./bin:$(PATH)
export NAME=linuxaid-cli
export VERSION=1.5.0
export DIST=./dist
export MAINTAINER=Ashish Jaiswal <ashish@obmondo.com>
export PREFIX=/opt/obmondo/bin/
SOURCES := linuxaid

.PHONY: all dep build clean test format vet lint

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

build: $(SOURCES)

$(SOURCES): dep
	CGO_ENABLED=0 go build -v -ldflags="-X main.Version=$(VERSION) -s -w" -o $(SOURCES)-install ./cmd/$(SOURCES)-install
	CGO_ENABLED=0 go build -v -ldflags="-X main.Version=$(VERSION) -s -w" -o $(SOURCES)-cli ./cmd/$(SOURCES)-cli

	chmod +x $(SOURCES)-install $(SOURCES)-cli