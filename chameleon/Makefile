.PHONY: build test clean run-parse run-validate

# Env variables for tests
TEST_DB_HOST ?= localhost
TEST_DB_PORT ?= 5433
TEST_DB_NAME ?= chameleon_test
TEST_DB_USER ?= postgres
TEST_DB_PASS ?= postgres

export CHAMELEON_TEST_DB_HOST=$(TEST_DB_HOST)
export CHAMELEON_TEST_DB_PORT=$(TEST_DB_PORT)
export CHAMELEON_TEST_DB_NAME=$(TEST_DB_NAME)
export CHAMELEON_TEST_DB_USER=$(TEST_DB_USER)
export CHAMELEON_TEST_DB_PASS=$(TEST_DB_PASS)

# Paths
RUST_LIB_DIR := $(shell pwd)/../chameleon-core/target/release
RUST_INCLUDE_DIR := $(shell pwd)/../chameleon-core/include

# Build Rust core first
build-rust:
	cd ../chameleon-core && cargo build --release

# Build Go binary with explicit library path
build: build-rust
	CGO_LDFLAGS="-L$(RUST_LIB_DIR) -lchameleon_core -Wl,-rpath,$(RUST_LIB_DIR)" \
	CGO_CFLAGS="-I$(RUST_INCLUDE_DIR)" \
	go build -o bin/chameleon ./cmd/chameleon

# Run tests
test: build-rust
	CGO_LDFLAGS="-L$(RUST_LIB_DIR) -lchameleon_core" \
	CGO_CFLAGS="-I$(RUST_INCLUDE_DIR)" \
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) \
	go test -v ./...

.PHONY: test-integration
test-integration:
	@echo "Building Rust library..."
	cd ../chameleon-core && cargo build --release

	@echo "Starting test containers..."
	docker compose -f docker-compose.test.yml up -d

	@echo "Waiting for PostgreSQL..."
	@sleep 8

	@echo "Running integration tests..."
	CGO_LDFLAGS="-L$(shell pwd)/../chameleon-core/target/release -lchameleon_core" \
	CGO_CFLAGS="-I$(shell pwd)/../chameleon-core/include" \
	LD_LIBRARY_PATH=$(shell pwd)/../chameleon-core/target/release \
	go test -v ./tests/integration/... -timeout 60s || \
	(docker compose -f docker-compose.test.yml down && exit 1)

	@echo "Stopping test containers..."
	docker compose -f docker-compose.test.yml down


.PHONY: test-all
test-all: test test-integration

.PHONY: docker-up
docker-up:
	docker compose -f docker-compose.test.yml up -d

.PHONY: docker-down
docker-down:
	docker compose -f docker-compose.test.yml down -v

# Clean builds
clean:
	rm -rf bin/
	cd ../chameleon-core && cargo clean

# Run examples (need LD_LIBRARY_PATH at runtime)
run-parse: build
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) ./bin/chameleon parse ../examples/basic_schema.cham

run-validate: build
	LD_LIBRARY_PATH=$(RUST_LIB_DIR) ./bin/chameleon validate ../examples/basic_schema.cham

# Install locally
install: build
	cp bin/chameleon $(GOPATH)/bin/