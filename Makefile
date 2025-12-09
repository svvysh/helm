# Helm TUI build helpers

SHELL := /bin/bash
ARGS ?=

.PHONY: setup fmt lint test run clean all release

setup:
	rustup toolchain install stable
	rustup component add rustfmt clippy
	git submodule update --init --recursive

fmt:
	cargo fmt --all

lint:
	cargo clippy --all-targets --all-features -D warnings

test:
	cargo test --all

run:
	cargo run --bin helm -- $(ARGS)

clean:
	cargo clean

all: setup fmt lint test

release:
	cargo build --release --locked
