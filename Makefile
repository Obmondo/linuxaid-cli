PROJECT_NAME := "go-scripts"
# PKG := "gitlab.enableit.dk/obmondo/monitoring/$(PROJECT_NAME)"

.PHONY: all dep build clean test format vet lint run

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

SOURCES := system_update

system_update: dep
	CGO_ENABLED=0 go build -a -v -ldflags="-extldflags=-static" -o ./cmd/system_update/$(SOURCES) ./cmd/system_update
	chmod +x ./cmd/system_update/$(SOURCES)


clean: ## Remove previous build
	@go clean

run:
	@go run ./cmd
