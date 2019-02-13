# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=go-test-rp
BINARY_UNIX=$(BINARY_NAME)_unix

packages := $(shell go list ./... | grep -v /vendor/)

all: test build

.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v

#mock:
#	go get github.com/vektra/mockery/.../
#	mockery -dir integration -output integration/mocks -name Clientable

.PHONY: test
test: clean
	$(GOTEST) -v $(packages) -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

.PHONY: run
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

.PHONY: lint
lint:
	go vet $(packages)

.PHONY: cover
cover:
	go tool cover -html=all.out -o all.html

