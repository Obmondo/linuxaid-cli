# Makefile for TLS Common Name Verifier

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build

# Binary name
BINARY_NAME = tls-common-name-verifier

all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) main.go

run:
	$(GOBUILD) -o $(BINARY_NAME) main.go
	./$(BINARY_NAME)
