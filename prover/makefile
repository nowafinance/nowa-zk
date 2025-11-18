# Simple Makefile for building, testing, and running a prover CLI.
MOD := github.com/tannetwork/zk-sequencer/prover
GO := go

.PHONY: all deps test build prove clean

all: test

# fetch gnark + other deps (idempotent)
deps:
	@echo "Fetching dependencies..."
	$(GO) get github.com/consensys/gnark@latest
	$(GO) mod tidy

# run unit tests for the prover package (compiles gnark deps)
test: deps
	$(GO) test ./prover/...

# build CLI
build:
	$(GO) build -o bin/prover ./cmd/prover

# run a prove step using the CLI (example)
# Provide any args as PROVE_ARGS="--circuit circuits/simple"
prove: build
	./bin/prover $(PROVE_ARGS)

clean:
	-rm -rf bin
	$(GO) clean
