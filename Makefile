PATH := ./bin:$(PATH)
export NAME=linuxaid-cli
export VERSION=$(shell ./scripts/latest-tag.sh)
export MAINTAINER=Ashish Jaiswal <ashish@obmondo.com>
SOURCES := linuxaid

.PHONY: all dep build clean test format vet lint release

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

.PHONY: release
release: ## Cut a new release using cocogitto (usage: make release [patch|minor|major|--auto])
	@if [ "$(shell git branch --show-current)" != "main" ]; then \
		echo "❌ Releases must be cut from 'main'"; exit 1; \
	fi
	@git fetch github --tags --force
	@git fetch origin --tags --force
	@git pull origin main --rebase
	@cog bump $(if $(filter-out $@,$(MAKECMDGOALS)),$(filter-out $@,$(MAKECMDGOALS)),--auto)
	@echo "✅ Successfully bumped version and pushed tags to both remotes (github & origin)!"