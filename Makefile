PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
TOOLS_MODULE := $(PROJECT_DIR)/tools.mod

# DEVELOPMENT
.PHONY: setup build install check lint tidy

SUBMODULES := $(shell find . -mindepth 2 -name go.mod -exec dirname {} \;)

setup: ##@development Setup the project and update dependencies
	go mod tidy
	@for mod in $(SUBMODULES); do (cd $$mod && go mod tidy); done

build: ##@development Build all packages
	go build ./...
	@for mod in $(SUBMODULES); do (cd $$mod && go build ./...); done

install: ##@development Install the xdb CLI binary
	cd cmd/xdb && go install .

check: ##@development Runs linting and formatting
check: tidy lint

lint: golangci-lint ##@development Runs golangci-lint (includes formatting, vetting, and linting)
	$(GOLANGCI_LINT) run --fix --config $(PROJECT_DIR)/.golangci.yml ./...
	@for mod in $(SUBMODULES); do (cd $$mod && $(GOLANGCI_LINT) run --fix --config $(PROJECT_DIR)/.golangci.yml ./...); done

tidy: ##@development Runs go mod tidy to update dependencies
	go mod tidy
	@for mod in $(SUBMODULES); do (cd $$mod && go mod tidy); done

# TESTING

.PHONY: test bench evals

test: ##@testing Run all tests
	go test -race -timeout=5m -covermode=atomic -coverprofile=coverage.out ./...
	@for mod in $(SUBMODULES); do echo "==> testing $$mod" && (cd $$mod && go test -race -timeout=5m ./...) || exit 1; done

evals: ##@testing Run the agent task evals (TASK=name[,name] MODEL=slug RUBRIC=1)
	cd cmd/xdb && go build -o ../../bin/xdb .
	cd internal/evals && go run . -tasks tasks $(if $(RESULTS),-results $(RESULTS)) -task "$(TASK)" -model "$(MODEL)" -binary $(PROJECT_DIR)/bin/xdb $(if $(RUBRIC),-rubric)

bench: ##@testing Run all benchmarks
	go test -bench=. -benchmem -run=^$$ -timeout=10m ./...
	@for mod in $(SUBMODULES); do echo "==> bench $$mod" && (cd $$mod && go test -bench=. -benchmem -run=^$$ -timeout=10m ./...) || exit 1; done

# SERVICES

.PHONY: services-up services-down services-logs

REDIS_CONTAINER := xdb-redis
REDIS_IMAGE := redis:8-alpine

services-up: ##@services Start service containers (Apple container)
	@container system start >/dev/null 2>&1 || true
	@container rm -f $(REDIS_CONTAINER) >/dev/null 2>&1 || true
	container run -d --name $(REDIS_CONTAINER) -p 6379:6379 $(REDIS_IMAGE)

services-down: ##@services Stop and remove service containers
	@container rm -f $(REDIS_CONTAINER) >/dev/null 2>&1 || true

services-logs: ##@services Tail service container logs
	container logs -f $(REDIS_CONTAINER)

# SITE

.PHONY: site-install site-dev site-build site-preview site-check docs-links

site-install: ##@site Install the site dependencies with pnpm
	cd site && pnpm install

site-dev: ##@site Start the site dev server
	cd site && pnpm dev

site-build: ##@site Build the site to site/dist
	cd site && pnpm build

site-preview: ##@site Serve the built site
	cd site && pnpm preview

site-check: ##@site Type-check the site
	cd site && pnpm check

docs-links: ##@site Find dead and site-absolute links in docs/
	node site/scripts/check-links.mjs docs

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
