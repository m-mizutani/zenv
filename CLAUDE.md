# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`zenv` is an enhanced `env` command to manage environment variables in CLI. It's a Go-based tool that:
- Loads environment variables from `.env` files (static values, file content, command execution)
- Securely manages secrets using macOS Keychain
- Replaces command arguments with environment variables

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

## Build and Test Commands

```bash
# Build the project
go build .

# Run tests
go test ./...

# Run go vet for static analysis
go vet ./...

# Run a single test
go test ./pkg/... -run TestName
```

## Architecture

The project is currently undergoing a rebuild (branch: rebuild/v2). The architecture follows a clean architecture pattern:

- `main.go`: Entry point that calls `cli.Run()`
- `pkg/cli/`: CLI interface layer (currently minimal implementation)
- Previous structure (being refactored):
  - `pkg/controller/`: Command handling
  - `pkg/domain/`: Core business logic and models
  - `pkg/infra/`: External dependencies (file system, keychain, execution)
  - `pkg/usecase/`: Application use cases

## Key Features to Maintain

1. **Environment Loading Priority**: `-e` options → additional `-e` options → command arguments
2. **Secret Management**: Namespaces must have `@` prefix
3. **Variable Replacement**: Words with `%` prefix are replaced with environment variables
4. **File Content Loading**: `&` prefix loads file content as variable value
5. **Command Execution**: Backticks execute commands and use output as variable value

## Development Notes

- Main branch for PRs: `main`
- Current working branch: `rebuild/v2`
- Significant refactoring in progress - many files have been deleted and new structure is being built in `pkg/cli/`