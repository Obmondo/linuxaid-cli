PATH := ../../bin:$(PATH)
export NAME=linuxaid-spray
export RELEASE=1.0.0
export DIST=../../dist
export VERSION=1.0.1
export MAINTAINER=Ashish Jaiswal <ashish@obmondo.com>
export PREFIX=/opt/obmondo/bin/
SOURCES := linuxaid-spray

.PHONY: all packages dep build deb deb_arm rpm

all: build

packages: deb rpm

test: ## Run unittests
	@go test -v

dep: ## Get the dependencies
	@go get -v ./...

build: $(SOURCES)

$(SOURCES): dep
	CGO_ENABLED=0 GOARCH=amd64 go build -a -v -ldflags="-X main.Version=$(VERSION) -extldflags=-static" -o amd64/$(SOURCES) .
	CGO_ENABLED=0 GOARCH=arm64 go build -a -v -ldflags="-X main.Version=$(VERSION) -extldflags=-static" -o arm64/$(SOURCES) .
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
