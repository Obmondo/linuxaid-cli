PATH := ../../bin:$(PATH)
export NAME=linuxaid-spray
export RELEASE=1.0.0
export VERSION=1.0.0
export MAINTAINER=Ashish Jaiswal <ashish@obmondo.com>
SOURCES := linuxaid-spray

test: ## Run unittests
	@go test -v

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -v -ldflags="-extldflags=-static" -o $(SOURCES)_amd64 $(SOURCES).go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -v -ldflags="-extldflags=-static" -o $(SOURCES)_arm64 $(SOURCES).go
	chmod +x $(SOURCES)_amd64
	chmod +x $(SOURCES)_arm64
	mv $(SOURCES)_amd64 $(SOURCES)
