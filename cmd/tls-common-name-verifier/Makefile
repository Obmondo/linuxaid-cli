# Makefile for TLS Common Name Verifier

PACKAGE_NAME = tls-common-name-verifier
PACKAGE_VERSION = 1.0.0
SOURCE_DIR = ./
PACKAGE_DIR = ./dist
BINARY_DIR = ./bin
PREFIX = /usr/local/bin
CONFIG_FILE=./opt/obmondo/etc/tls-common-name-verifier/config.yaml
EXTERNAL_FACT=./opt/puppetlabs/facter/facts.d/tls-common-name-verifier.sh

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build

# Binary name
BINARY_NAME = tls-common-name-verifier

packages: build deb rpm

all: build

build:
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) main.go

run: build
	./$(BINARY_DIR)/$(BINARY_NAME)

deb:
	@echo "Building Debian package..."
	@mkdir -p $(PACKAGE_DIR)
	@fpm -s dir -t deb -n $(PACKAGE_NAME) -v $(PACKAGE_VERSION) -C $(BINARY_DIR) \
	  -p $(PACKAGE_DIR) -a all --prefix=${PREFIX} \
		--config-files ${CONFIG_FILE} \
		--config-files ${EXTERNAL_FACT} \
		$(BINARY_NAME)

rpm:
	@echo "Building RPM package..."
	@mkdir -p $(PACKAGE_DIR)
	@fpm -s dir -t rpm -n $(PACKAGE_NAME) -v $(PACKAGE_VERSION) -C $(BINARY_DIR) \
	  -p $(PACKAGE_DIR) -a all --prefix=${PREFIX} \
		--config-files ${CONFIG_FILE} \
		--config-files ${EXTERNAL_FACT} \
		$(BINARY_NAME)

clean:
	@echo "Cleaning up..."
	@rm -rf $(PACKAGE_DIR) $(PACKAGE_DIR)

.PHONY: all deb rpm clean
