# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based image labeling application. The project is in early development stages with the following structure:
- `main.go` - Entry point for the application
- `images/` - Directory for image storage/processing
- `website/` - Directory for web interface components

## Common Commands

### Building and Running
```bash
go run main.go          # Run the application
go build                # Build the binary
go build -o image-label # Build with custom output name
```

### Development
```bash
go fmt ./...            # Format all Go files
go vet ./...            # Run Go vet for static analysis
go test ./...           # Run all tests
go test -v ./...        # Run tests with verbose output
go test -run TestName   # Run a specific test
```

### Dependencies
```bash
go mod tidy             # Clean up dependencies
go mod download         # Download dependencies
go get <package>        # Add a new dependency
```

## Architecture Notes

The project uses Go 1.25.4 and is structured as a single module at `github.com/harveytvt/image-label`.

The application appears designed to handle image labeling with both a backend (main Go application) and frontend (website directory) component.
