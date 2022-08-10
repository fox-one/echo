PROJECT_NAME := "github.com/fox-one/echo"
PKG := "$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep lint vet test test-coverage echob echos build clean

all: build

dep: ## Get the dependencies
	@go mod download

lint: ## Lint Golang files
	@golangci-lint run

vet: ## Run go vet
	@go vet ${PKG_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

test-coverage: ## Run tests with coverage
	@go test -short -coverprofile cover.out -covermode=atomic ${PKG_LIST}
	@cat cover.out >> coverage.txt

echob:
	@go build -o build/echob ./cmd/echob

echos:
	@go build -o build/echos ./cmd/echos

scanner:
	@go build -o build/scanner ./cmd/scanner

build: dep echob echos scanner

clean: ## Remove previous build
	@rm -rf ./build
