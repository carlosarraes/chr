# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`chr` is a command-line tool for managing Git branch commits and cherry-picking between production (PRD) and homologation (HML) branches. It solves the "rebase problem" by using composite commit matching (author + date + message) instead of relying on Git hashes.

## Build and Test Commands

```bash
# Build the binary
make build

# Run tests with coverage
make test
make test-coverage

# Format and lint code
make fmt
make lint

# Development build with race detection
make dev-build

# Run the tool directly
make run

# Install locally
make install-local
```

## Architecture

The codebase follows a clean architecture pattern with these key components:

- **main.go**: Entry point that delegates to cmd package
- **cmd/**: CLI command handling using Kong framework
  - `cli.go`: Main CLI structure and command implementations
  - `root.go`: Command execution wrapper
- **internal/config/**: Configuration management using Koanf
  - Supports TOML config files, environment variables, and defaults
  - Config path: `~/.config/chr.toml`
- **internal/git/**: Git operations and commit handling
- **internal/picker/**: Commit filtering and matching logic

## Key Design Patterns

1. **Composite Commit Matching**: Uses author name + commit date + first line of message to identify commits across rebases
2. **Layered Configuration**: Defaults → config file → environment variables → CLI flags
3. **Branch Naming Convention**: `{prefix}{card-number}{suffix}` pattern (e.g., `ZUP-123-prd`, `ZUP-123-hml`)

## Development Notes

- Uses Go 1.21+ with Kong CLI framework for command parsing
- Configuration managed via Koanf library with TOML format
- Colored output using fatih/color library
- All tests should be placed alongside source files with `_test.go` suffix
- The tool expects to run inside Git repositories and follows conventional commit patterns

## Environment Variables

- `CHR_PREFIX`: Branch name prefix override
- `CHR_SUFFIX_PRD`: Production branch suffix override  
- `CHR_SUFFIX_HML`: Homologation branch suffix override
- `CHR_COLOR`: Enable/disable colored output (true/false)