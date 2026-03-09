# Wikipedia CLI

A command-line Wikipedia client built with the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework.

## Features
- Search Wikipedia articles directly from the terminal
- Read article content without leaving the CLI

## Requirements
- Go 1.26.1 or newer

## Install with Homebrew
Install via the custom tap:

```bash
brew tap vrindavansanap/tap
brew install wikipedia-cli
```

## Setup
Clone the repository and install dependencies:

```bash
git clone <repo-url>
cd wikipedia-cli-2
go mod download
```

## Running
Run the app without building a binary:

```bash
go run .
```

Build and run a compiled binary via the included make target:

```bash
make run
```

If you only want the binary, use:

```bash
make build
./bin/wikipedia-cli
```

## Cleaning Up
Remove the generated binary and build artifacts:

```bash
make clean
```
