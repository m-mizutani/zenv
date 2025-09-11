# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`zenv` v2 is an enhanced `env` command to manage environment variables in CLI. It's a Go-based tool that:
- Loads environment variables from multiple sources with clear precedence
- Supports both `.env` files (simple key-value pairs) and TOML files (advanced configuration)
- Allows file content reading and command execution for dynamic values
- Supports inline environment variable specification (KEY=value format)
- Replaces command arguments with environment variables using `%` prefix

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

The v2 rebuild has been completed with a clean architecture pattern:

- `main.go`: Entry point that calls `cli.Run()`
- `pkg/cli/`: CLI interface layer - handles command line parsing and execution
- `pkg/usecase/`: Application use cases - orchestrates business logic
- `pkg/loader/`: Environment variable loaders
  - `dotenv_loader.go`: Loads variables from .env files
  - `toml_loader.go`: Loads variables from TOML files with advanced features
- `pkg/executor/`: Command execution layer
  - `executor.go`: Interface for command execution
  - `default_executor.go`: Default implementation using os/exec
- `pkg/model/`: Core domain models
  - `config.go`: Configuration structures
  - `env.go`: Environment variable models

## Key Features

1. **Environment Variable Priority**: System < .env < TOML < Inline (later sources override earlier ones)
2. **Variable Replacement**: Words with `%` prefix in command arguments are replaced with environment variables
3. **TOML Configuration**: Supports three modes for each variable:
   - `value`: Direct string value (supports multiline)
   - `file`: Load content from file
   - `command` + `args`: Execute command and use output as value
4. **.env File Format**: Simple KEY=VALUE pairs with support for:
   - File content loading: `KEY=&/path/to/file`
   - Command execution: `` KEY=`command` ``
   - Comments: Lines starting with `#`

## Development Notes

- Main branch for PRs: `main`
- Version: v2 (major rewrite completed)
- Migration guide available at `docs/migration.md` for users upgrading from v1