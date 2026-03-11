PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
TOOLS_MODULE := $(PROJECT_DIR)/tools.mod

# DEVELOPMENT
.PHONY: setup build check lint tidy

SUBMODULES := $(shell find . -mindepth 2 -name go.mod -exec dirname {} \;)

setup: ##@development Setup the project and update dependencies
	go mod tidy
	@for mod in $(SUBMODULES); do (cd $$mod && go mod tidy); done

build: ##@development Build all packages
	go build ./...
	@for mod in $(SUBMODULES); do (cd $$mod && go build ./...); done

check: ##@development Runs linting and formatting
check: tidy lint

lint: golangci-lint ##@development Runs golangci-lint (includes formatting, vetting, and linting)
	$(GOLANGCI_LINT) run --fix --config $(PROJECT_DIR)/.golangci.yml ./...
	@for mod in $(SUBMODULES); do (cd $$mod && $(GOLANGCI_LINT) run --fix --config $(PROJECT_DIR)/.golangci.yml ./...); done

tidy: ##@development Runs go mod tidy to update dependencies
	go mod tidy
	@for mod in $(SUBMODULES); do (cd $$mod && go mod tidy); done

# TESTING

.PHONY: test test-redis

test: ##@testing Run tests
	go test -race -timeout=5m -covermode=atomic -coverprofile=coverage.out ./...

test-redis: ##@testing Run Redis integration tests (requires running Redis)
	cd store/xdbredis && go test -race -timeout=5m ./...

# COVERAGE

.PHONY: coverage report

coverage: ##@tests Generates coverage report
	$(GOCOV) convert coverage.out > coverage.json
	$(GOCOV) convert coverage.out | $(GOCOV) report

report: coverage ##@tests Generates html coverage report
	$(GOCOVHTML) -t kit < coverage.json > coverage.html
	@open coverage.html

# TOOLS

GOLANGCI_LINT = go tool -modfile=$(TOOLS_MODULE) golangci-lint
golangci-lint:
	$(call go-get-tool,github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.0)

GOCOV = go tool -modfile=$(TOOLS_MODULE) gocov
gocov:
	$(call go-get-tool,github.com/axw/gocov/gocov@v1.1.0)

GOCOVHTML = go tool -modfile=$(TOOLS_MODULE) gocov-html
gocov-html:
	$(call go-get-tool,github.com/matm/gocov-html/cmd/gocov-html@v1.4.0)

# go-get-tool will 'go get -tool' any package $1
define go-get-tool
{ \
set -e ;\
go get -modfile=$(TOOLS_MODULE) -tool $(1) ;\
}
endef
