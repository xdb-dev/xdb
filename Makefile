GO_BUILD_DIRS := $(shell find . -type f -name 'go.mod' -not -path "*/example*" -exec dirname {} \; | sort)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
TOOLS_MODULE := $(PROJECT_DIR)/tools.mod

# Export GOEXPERIMENT=jsonv2 for all make commands
export GOEXPERIMENT=jsonv2

# DEVELOPMENT
.PHONY: setup lint check tidy

setup: ##@development Setup the project and update dependencies
	go mod tidy

check: ##@development Runs linting and formatting
check: tidy lint

lint: golangci-lint ##@development Runs golangci-lint (includes formatting, vetting, and linting)
	@$(call run-go-mod-dir,$(GOLANGCI_LINT) run --fix --config $(PROJECT_DIR)/.golangci.yml ./...,"golangci-lint")

tidy: ##@development Runs go mod tidy to update dependencies
	@$(call run-go-mod-dir,go mod tidy,"go mod tidy")

# TESTING

.PHONY: test

test: ##@testing Run tests
	@$(call run-go-mod-dir,go test -race -timeout=5m -covermode=atomic -coverprofile=coverage.out ./...,"go test")

# COVERAGE

.PHONY: coverage

coverage: ##@tests Generates coverage report
	@$(call run-go-mod-dir,$(GOCOV) convert coverage.out > coverage.json)
	@$(call run-go-mod-dir,$(GOCOV) convert coverage.out | $(GOCOV) report)

report: coverage ##@tests Generates html coverage report
	@jq -n '{ Packages: [ inputs.Packages ] | add }' $(shell find . -type f -name 'coverage.json' | sort) | $(GOCOVHTML) -t kit > coverage.html
	@open coverage.html


# TOOLS

GOLANGCI_LINT = go tool -modfile=$(TOOLS_MODULE) golangci-lint
golangci-lint:
	$(call go-get-tool,github.com/golangci/golangci-lint/cmd/golangci-lint@latest)

GOCOV = go tool -modfile=$(TOOLS_MODULE) gocov
gocov:
	$(call go-get-tool,github.com/axw/gocov/gocov@v1.1.0)

GOCOVHTML = go tool -modfile=$(TOOLS_MODULE) gocov-html
gocov-html:
	$(call go-get-tool,github.com/matm/gocov-html/cmd/gocov-html@v1.4.0)
	
# go-get-tool will 'go get -tool' any package $2
define go-get-tool
{ \
set -e ;\
go get -modfile=$(TOOLS_MODULE) -tool $(1) ;\
}
endef

# run-go-mod-dir runs the given $1 command in all the directories with
# a go.mod file
define run-go-mod-dir
set -e; \
for dir in $(GO_BUILD_DIRS); do \
	[ -z $(2) ] || echo "$(2) $${dir}/..."; \
	cd "$(PROJECT_DIR)/$${dir}" && PATH=$(BIN_DIR):$$PATH $(1); \
done;
endef

# run-go-mod-dir-exclude runs the given $1 command in all the directories with
# a go.mod file except the directories in $2
define run-go-mod-dir-exclude
set -e; \
for dir in $(filter-out $(2),$(GO_BUILD_DIRS)); do \
	[ -z $(3) ] || echo "$(3) $${dir}/..."; \
	cd "$(PROJECT_DIR)/$${dir}" && PATH=$(BIN_DIR):$$PATH $(1); \
done;
endef
