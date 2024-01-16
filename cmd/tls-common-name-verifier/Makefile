# Makefile for TLS Common Name Verifier

PACKAGE_NAME = tls-common-name-verifier
PACKAGE_VERSION = 1.0.0
SOURCE_DIR = ./
DEB_PACKAGE_DIR = ./build/deb
RPM_PACKAGE_DIR = ./build/rpm

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build

# Binary name
BINARY_NAME = tls-common-name-verifier

packages: deb rpm

all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) main.go

run:
	$(GOBUILD) -o $(BINARY_NAME) main.go
	./$(BINARY_NAME)

deb:
	@echo "Building Debian package..."
	@mkdir -p $(DEB_PACKAGE_DIR)
	@fpm -s dir -t deb -n $(PACKAGE_NAME) -v $(PACKAGE_VERSION) -C $(SOURCE_DIR) .

rpm:
	@echo "Building RPM package..."
	@mkdir -p $(RPM_PACKAGE_DIR)
	@fpm -s dir -t rpm -n $(PACKAGE_NAME) -v $(PACKAGE_VERSION) -C $(SOURCE_DIR) .

clean:
	@echo "Cleaning up..."
	@rm -rf $(DEB_PACKAGE_DIR) $(RPM_PACKAGE_DIR)

.PHONY: all deb rpm clean
