DEBUG_PORT?=2345
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTOOL=$(GOCMD) tool
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run
GOGET=$(GOCMD) get
GODEBUG=$(GOTOOL) dlv dap --log --listen :${DEBUG_PORT}
GOGENERATE=$(GOCMD) generate
COVERAGE_EXCLUDE_FILES=cat coverprofile.out | grep -v "_mock.go" | grep -v "mocks.go" | grep -v "_test.go" | grep -v "test/" | grep -v "configs/"
BIN_FOLDER=bin

export GOMAXPROCS=5
export GO111MODULE=on

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: init
init:
ifeq (,$(wildcard ./.env))
	@make env-file
endif
	@make clean deps docker-up run

env-file:
	@cp ./.env.example ./.env

.PHONY: build
build: deps
	@$(GOBUILD) -mod vendor -a -o $(BIN_FOLDER)/app ./cmd

.PHONY: test
test:
	@$(GOTEST) -cover --coverpkg=./... ./...
	@$(COVERAGE_EXCLUDE_FILES) > temp.coverprofile.out
	@cat temp.coverprofile.out > coverprofile.out
	@rm -r temp.coverprofile.out

test-bench:
	@$(GOTEST) --cpuprofile cpuprofile.out --memprofile memprofile.out

test-cover: test
	@$(GOTOOL) cover -html=coverprofile.out

.PHONY: test-coer-ci
test-cover-xml: test
	@$(GOTOOL) gocov convert coverprofile.out | gocov-xml > coverage.xml

clean:
	@$(GOCLEAN) -cache -modcache -i -r # optional
	@rm -f $(BINARY_NAME)

.PHONY: run
run: 
	@$(GORUN) cmd/*.go

run-debug: 
	@$(GODEBUG) cmd/*.go

deps:
	@$(GOMOD) vendor -v

deps-check:
	@$(GOMOD) tidy -v

deps-upgrade:
	@$(GOGET) -u ./...
	@$(GOMOD) tidy -v
	@$(GOMOD) vendor -v

lint:
	@$(GOTOOL) golangci-lint run --modules-download-mode vendor --timeout 2m

lint-fix:
	@$(GOTOOL) golangci-lint run --fix

docker-%:
	@make -C docker/ $*