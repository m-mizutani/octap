# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**octap** - A CLI GitHub Actions notifier written in Go.

## Restriction & Rules

- In principle, do not trust developers who use this library from outside
  - Do not export unnecessary methods, structs, and variables
  - Assume that exposed items will be changed. Never expose fields that would be problematic if changed
  - Use `export_test.go` for items that need to be exposed for testing purposes
- When making changes, before finishing the task, always:
  - Run `go vet ./...`, `go fmt ./...` to format the code
  - Run `golangci-lint run ./...` to check lint error
  - Run `gosec -quiet ./...` to check security issue
  - Run tests to ensure no impact on other code
- All comment and character literal in source code must be in English
- Test files should have `package {name}_test`. Do not use same package name
- Test must be included in same name test file. (e.g. test for `abc.go` must be in `abc_test.go`)
- Use named empty structure (e.g. `type ctxHogeKey struct{}` ) as private context key
- Do not create binary. If you need to run, use `go run` command instead
- When a `tmp` directory is specified, search for files within the `./tmp` directory relative to the project root.

## Tools & Libraries

You must use following tools and libraries for development.

- logging: Use `log/slog`. If you need to decorate logging message, use `github.com/m-mizutani/clog`
- CLI framework: `github.com/urfave/cli/v3`
- Error handling: `github.com/m-mizutani/goerr/v2`
- Testing framework: `github.com/m-mizutani/gt`
- Logger propagation: `github.com/m-mizutani/ctxlog`
- Task management: `https://github.com/go-task/task`
- Mock generation: `github.com/matryer/moq` for interface mocking

## Common Development Commands

### Go Module Management
- `go mod init` - Initialize Go module (already done)
- `go mod tidy` - Clean up and download dependencies
- `go get <package>` - Add new dependencies

### Building and Running
- `go build` - Build the binary
- `go run .` or `go run main.go` - Run the application directly
- `go build -o octap` - Build with specific output name

### Testing
- `go test ./...` - Run all tests
- `go test -v ./...` - Run tests with verbose output
- `go test -cover ./...` - Run tests with coverage

### Code Quality
- `go fmt ./...` - Format all Go files
- `go vet ./...` - Run static analysis
- `golangci-lint run` - Run linter (if installed)

## Architecture Notes

This is a Go CLI application for GitHub Actions notifications. The project is currently in initial setup phase with Go 1.25.0 as specified in go.mod.

When developing:
- Follow standard Go project layout conventions
- Place main package in root or cmd/octap/
- Use internal/ for private application code
- Use pkg/ for library code that could be imported by external projects