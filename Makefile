TOOLS_DIR := internal/tools
GO_BUILD_DIRS := $(shell find . -type f -name 'go.mod' -not -path "./internal/tools/*" -not -path "*/example*" -exec dirname {} \; | sort)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

# DEVELOPMENT
.PHONY: setup fmt vet lint check imports tidy

setup: ##@development Setup the project
	go mod tidy
	cd $(TOOLS_DIR) && go mod tidy

check: ##@development Runs formatting, vetting and linting
check: tidy
check: imports
check: fmt vet lint

fmt:
	@$(call run-go-mod-dir,go fmt ./...,"go fmt")

vet:
	@$(call run-go-mod-dir,go vet ./...,"go vet")

lint:
	@$(call run-go-mod-dir,$(REVIVE) -config $(PROJECT_DIR)/revive.toml ./...,"revive")

imports:
	@$(call run-go-mod-dir,$(GCI) -w -local github.com/xdb-dev/xdb ./ | { grep -v -e 'skip file .*' || true; },"gci")

tidy:
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

GCI = go tool -modfile=$(TOOLS_DIR)/go.mod gci
gci:
	$(call go-get-tool,github.com/daixiang0/gci@v0.2.9)

REVIVE = go tool -modfile=$(TOOLS_DIR)/go.mod revive
revive:
	$(call go-get-tool,github.com/mgechev/revive@latest)

GOCOV = go tool -modfile=$(TOOLS_DIR)/go.mod gocov
gocov:
	$(call go-get-tool,github.com/axw/gocov/gocov@v1.1.0)

GOCOVHTML = go tool -modfile=$(TOOLS_DIR)/go.mod gocov-html
gocov-html:
	$(call go-get-tool,github.com/matm/gocov-html/cmd/gocov-html@v1.4.0)
	
# go-get-tool will 'go get -tool' any package $2
define go-get-tool
{ \
set -e ;\
cd $(TOOLS_DIR);\
go get -tool $(1) ;\
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
