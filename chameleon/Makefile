.PHONY: build test clean run-parse run-validate

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