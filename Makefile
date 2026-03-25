GO ?= $(shell which go 2>/dev/null)

BINARY_NAME := docker-info
BUILD_PATH := build
BINARY_PATH := $(BUILD_PATH)/$(BINARY_NAME)
INSTALL_PATH := /usr/local/bin/$(BINARY_NAME)

.PHONY: all build clean install uninstall help

all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) using $(GO)..."
	mkdir -p $(BUILD_PATH)
	@$(GO) build -o $(BINARY_PATH) main.go

## clean: Remove binary and build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_PATH)

## install: Build and install binary system-wide
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_PATH) $(INSTALL_PATH)
	@echo "Done! You can now run '$(BINARY_NAME)' from anywhere."

## uninstall: Remove binary from the system
uninstall:
	@echo "Removing $(BINARY_NAME) from $(INSTALL_PATH)..."
	@sudo rm -f $(INSTALL_PATH)
