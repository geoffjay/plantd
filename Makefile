M := $(shell printf "\033[34;1m▶\033[0m")
VERSION := $(shell git describe 2>/dev/null || echo "undefined")
SHELL := /bin/bash
BUILD_ARGS := -ldflags "-X core.VERSION=$(VERSION)"
TEST_ARGS := $(shell if [ ! -z ${COVERAGE} ]; then echo "-race -coverprofile=coverage.txt -covermode=atomic"; fi)

all: build

-include init/setup.mk
-include logger/manage.mk

setup: deps hooks

deps: ; $(info $(M) Installing dependencies...)
	@./scripts/install-deps

hooks: ; $(info $(M) Installing commit hooks...)
	@./scripts/install-hooks

proto-gen: ; $(info $(M) Generating protocol buffers...)
	@PATH="$$HOME/.asdf/shims:$$(go env GOPATH)/bin:$$PATH" buf generate

proto-lint: ; $(info $(M) Linting protocol buffers...)
	@PATH="$$HOME/.asdf/shims:$$(go env GOPATH)/bin:$$PATH" buf lint

proto-breaking: ; $(info $(M) Checking for breaking changes...)
	@PATH="$$HOME/.asdf/shims:$$(go env GOPATH)/bin:$$PATH" buf breaking --against '.git#branch=main'

proto-clean: ; $(info $(M) Cleaning generated protocol buffers...)
	@rm -rf gen/proto

# Add proto validation to CI
ci-proto: proto-lint proto-gen ; $(info $(M) Protocol buffer validation complete)
	@echo "Protocol buffer validation complete"

# Traefik gateway development
.PHONY: traefik-dev-start traefik-dev-stop traefik-dev-restart traefik-dev-status traefik-dev-test traefik-dev-logs

traefik-dev-start: ; $(info $(M) Starting Traefik gateway environment...)
	@./scripts/traefik-dev.sh start

traefik-dev-stop: ; $(info $(M) Stopping Traefik gateway environment...)
	@./scripts/traefik-dev.sh stop

traefik-dev-restart: ; $(info $(M) Restarting Traefik gateway environment...)
	@./scripts/traefik-dev.sh restart

traefik-dev-status: ; $(info $(M) Checking Traefik gateway status...)
	@./scripts/traefik-dev.sh status

traefik-dev-test: ; $(info $(M) Testing Traefik gateway functionality...)
	@./scripts/traefik-dev.sh test

traefik-dev-logs: ; $(info $(M) Showing Traefik gateway logs...)
	@./scripts/traefik-dev.sh logs

lint: ; $(info $(M) Lint projects...)
	@./scripts/utility go-lint app
	@./scripts/utility go-lint broker
	@./scripts/utility go-lint core
	@./scripts/utility go-lint identity
	@./scripts/utility go-lint logger
	@./scripts/utility go-lint proxy
	@./scripts/utility go-lint state

build: build-pre proto-gen build-app build-broker build-client build-identity build-logger build-proxy build-state

build-grpc: build-pre proto-gen build-state-grpc

build-all: build build-grpc build-client-grpc

build-pre: ; $(info $(M) Building projects...)
	@mkdir -p build/

build-app: ; $(info $(M) Building app service...)
	@pushd app >/dev/null; \
	go build -o ../build/plantd-app $(BUILD_ARGS) .; \
	popd >/dev/null

build-broker: ; $(info $(M) Building broker service...)
	@pushd broker >/dev/null; \
	go build -o ../build/plantd-broker $(BUILD_ARGS) .; \
	popd >/dev/null

build-client: ; $(info $(M) Building client...)
	@cd client && go build -o ../build/plant main.go

build-client-grpc: proto-gen client-deps ; $(info $(M) Building client with gRPC support...)
	@cd client && go mod tidy && go build -o ../build/plant-grpc main.go

client-deps: ; $(info $(M) Installing client dependencies...)
	@cd client && go mod tidy

build-identity: ; $(info $(M) Building identity service...)
	@pushd identity >/dev/null; \
	go build -o ../build/plantd-identity $(BUILD_ARGS) ./cmd/main.go; \
	popd >/dev/null

build-logger: ; $(info $(M) Building logger service...)
	@pushd logger >/dev/null; \
	go build -o ../build/plantd-logger $(BUILD_ARGS) .; \
	popd >/dev/null

build-proxy: ; $(info $(M) Building proxy service...)
	@pushd proxy >/dev/null; \
	go build -o ../build/plantd-proxy $(BUILD_ARGS) .; \
	popd >/dev/null

build-state: ; $(info $(M) Building state service...)
	@pushd state >/dev/null; \
	go build -o ../build/plantd-state $(BUILD_ARGS) .; \
	popd >/dev/null

build-state-grpc: ; $(info $(M) Building state gRPC service...)
	@pushd state >/dev/null; \
	go build -o ../build/plantd-state-grpc $(BUILD_ARGS) ./grpc_main.go ./grpc_server.go ./mdp_compat.go ./store.go; \
	popd >/dev/null

build-module-echo: ; $(info $(M) Building echo module...)
	@pushd module/echo >/dev/null; \
	go build -o ../../build/plantd-module-echo $(BUILD_ARGS) .; \
	popd >/dev/null

test: test-pre test-core test-broker test-identity test-state test-app

test-pre: ; $(info $(M) Testing projects...)
	@mkdir -p coverage/

test-integration:
	@pushd core >/dev/null; \
	go test --tags=integration ./... -v; \
	popd >/dev/null

test-core:
	@pushd core >/dev/null; \
	go test $(TEST_ARGS) ./... -v; \
	if [[ -f coverage.txt ]]; then mv coverage.txt ../coverage/core.txt; fi; \
	popd >/dev/null

test-broker:
	@pushd broker >/dev/null; \
	go test $(TEST_ARGS) ./... -v; \
	if [[ -f coverage.txt ]]; then mv coverage.txt ../coverage/broker.txt; fi; \
	popd >/dev/null

test-identity:
	@pushd identity >/dev/null; \
	go test $(TEST_ARGS) ./... -v; \
	if [[ -f coverage.txt ]]; then mv coverage.txt ../coverage/identity.txt; fi; \
	popd >/dev/null

test-state:
	@pushd state >/dev/null; \
	go test $(TEST_ARGS) ./... -v; \
	if [[ -f coverage.txt ]]; then mv coverage.txt ../coverage/state.txt; fi; \
	popd >/dev/null

test-app:
	@pushd app >/dev/null; \
	go test $(TEST_ARGS) ./... -v; \
	if [[ -f coverage.txt ]]; then mv coverage.txt ../coverage/app.txt; fi; \
	popd >/dev/null

test-e2e:
	@pushd app >/dev/null; \
	bun run test:e2e; \
	popd >/dev/null

# live reload helpers
dev:
	@overmind start

dev-app:
	@air -c app/.air.toml

dev-broker:
	@air -c broker/.air.toml

dev-identity:
	@air -c identity/.air.toml

dev-logger:
	@air -c logger/.air.toml

dev-proxy:
	@air -c proxy/.air.toml

dev-state:
	@air -c state/.air.toml

gen-app-apidocs:
	@pushd app >/dev/null; \
	swag init --dir "./,./handlers" -g main.go; \
	popd >/dev/null

# docker helpers
docker: docker-pre docker-broker docker-state docker-logger docker-proxy docker-module-echo

docker-pre: ; $(info $(M) Building docker images)

docker-broker:
	@docker build -t geoffjay/plantd-broker -f broker/Dockerfile .

docker-identity:
	@docker build -t geoffjay/plantd-identity -f identity/Dockerfile .

docker-logger:
	@docker build -t geoffjay/plantd-logger -f logger/Dockerfile .

docker-proxy:
	@docker build -t geoffjay/plantd-proxy -f proxy/Dockerfile .

docker-state:
	@docker build -t geoffjay/plantd-state -f state/Dockerfile .

docker-module-echo:
	@docker build -t geoffjay/plantd-module-echo -f module/echo/Dockerfile .

# notebooks for new ideas
jupyter:
	@mkdir -p notebooks
	@docker run -it -p 8888:8888 -v notebooks:/notebooks gopherdata/gophernotes:latest-ds

install: ; $(info $(M) Installing binaries...)
	@install build/plantd-* /usr/local/bin/

uninstall: ; $(info $(M) Uninstalling binaries...)
	@rm /usr/local/bin/plantd-*

clean: ; $(info $(M) Removing build files...)
	@rm -rf build/
	@rm -rf coverage/

# Test targets
.PHONY: test-grpc-client

test-grpc-client: build-client-grpc ; $(info $(M) Testing gRPC client implementation...)
	@chmod +x scripts/test-grpc-client.sh
	@./scripts/test-grpc-client.sh

# Phase 6: Integration Testing targets
.PHONY: test-integration test-load test-load-custom test-failure-scenarios test-migration-compatibility test-phase6

test-integration: build-client-grpc ; $(info $(M) Running integration tests...)
	@chmod +x scripts/test-integration.sh
	@./scripts/test-integration.sh

test-load: build-client-grpc ; $(info $(M) Running load tests...)
	@chmod +x scripts/test-load.sh
	@./scripts/test-load.sh

test-load-custom: build-client-grpc ; $(info $(M) Running custom load tests...)
	@chmod +x scripts/test-load.sh
	@./scripts/test-load.sh --users $(or $(USERS),10) --operations $(or $(OPERATIONS),100) --ramp-up $(or $(RAMP_UP),10) --think-time $(or $(THINK_TIME),0.1)

test-failure-scenarios: build-client-grpc ; $(info $(M) Running failure scenario tests...)
	@chmod +x scripts/test-failure-scenarios.sh
	@./scripts/test-failure-scenarios.sh

test-migration-compatibility: build-client-grpc ; $(info $(M) Testing migration compatibility...)
	@chmod +x scripts/test-migration-compatibility.sh
	@./scripts/test-migration-compatibility.sh

test-phase6: test-integration test-load test-failure-scenarios test-migration-compatibility ; $(info $(M) Phase 6 Integration Testing Complete)
	@echo ""
	@echo "Results available in:"
	@echo "  - test-results/integration/"
	@echo "  - test-results/load/"
	@echo "  - test-results/failure/"
	@echo "  - test-results/migration/"

.PHONY: all build clean
.PHONY: proto-gen proto-lint proto-breaking proto-clean ci-proto
