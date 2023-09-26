.PHONY: all dep build clean test format vet lint run

BINARY_NAME := kopsUpgrade

all: build

lint: ## Lint the files
	@golangci-lint run --issues-exit-code=1

format: ## Format the files
	@go fmt ./...

vet: ## Vet the files
	@go vet ./...


dep: ## Fetch the dependencies
	@go mod download

build: dep
	CGO_ENABLED=0 go build -a -v -ldflags="-extldflags=-static" -o ./$(BINARY_NAME) 


clean: ## Remove previous build
	@go clean

run: ## Run the kopsUpgrade.go file
	@go run kopsUpgrade.go