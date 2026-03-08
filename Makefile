# Variables
BINARY_NAME=wikipedia-cli
BIN_DIR=bin
BINARY_PATH=$(BIN_DIR)/$(BINARY_NAME)

# Default target
all: build

## build: Create the bin directory and build the binary into it
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY_PATH) main.go structs.go

## run: Build and run from the bin folder
run: build
	./$(BINARY_PATH)

## clean: Remove the entire bin directory
clean:
	rm -rf $(BIN_DIR)

.PHONY: all build run clean