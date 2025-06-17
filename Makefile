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

lint: ; $(info $(M) Lint projects...)
	@./scripts/utility go-lint app
	@./scripts/utility go-lint broker
	@./scripts/utility go-lint core
	@./scripts/utility go-lint identity
	@./scripts/utility go-lint logger
	@./scripts/utility go-lint proxy
	@./scripts/utility go-lint state

build: build-pre build-app build-broker build-client build-identity build-logger build-proxy build-state

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

build-client: ; $(info $(M) Building client utlity...)
	@pushd client >/dev/null; \
	go build -o ../build/plant $(BUILD_ARGS) .; \
	popd >/dev/null

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

build-module-echo: ; $(info $(M) Building echo module...)
	@pushd module/echo >/dev/null; \
	go build -o ../../build/plantd-module-echo $(BUILD_ARGS) .; \
	popd >/dev/null

test: test-pre test-core test-broker test-identity test-state

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

.PHONY: all build clean
