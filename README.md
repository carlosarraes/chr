# chr

A command-line tool for managing Git branch commits and cherry-picking between production and homologation branches.

## Overview

`chr` helps developers manage commits between production (PRD) and homologation (HML) branches in Git workflows. It intelligently identifies commits that need to be cherry-picked and handles the rebase-safe commit matching problem.

## Key Features

- **Rebase-Safe Commit Detection**: Uses composite matching (author + date + message) to identify commits even after rebases change their hashes
- **User-Friendly CLI**: Clean `chr pick` interface with intuitive flags
- **Automatic User Filtering**: `--latest` flag filters commits by current Git user
- **Date-Based Filtering**: Filter commits by today, yesterday, or custom date ranges
- **Dry-Run Mode**: `--show` flag previews commits before cherry-picking
- **Colored Output**: Syntax highlighting for different commit types and authors
- **Configurable Branch Naming**: Support for custom prefixes and suffixes
- **AI-Friendly Documentation**: `--llm` flag provides comprehensive usage guide for LLMs

## Installation

### Quick Install (Recommended)

```bash
curl -sSf https://raw.githubusercontent.com/carlosarraes/chr/main/install.sh | sh
```

This will install `chr` to `~/.local/bin`. Make sure this directory is in your PATH.

### Manual Installation

Download the latest binary from the [releases page](https://github.com/carlosarraes/chr/releases) and place it in your PATH.

### From Source

```bash
git clone https://github.com/carlosarraes/chr
cd chr
make build
sudo cp dist/chr /usr/local/bin/
```

Or install locally:

```bash
make install-local
```

## Usage

### Basic Usage

```bash
# Cherry-pick commits (default behavior)
chr pick

# Show commits without picking (dry-run)
chr pick --show

# Pick latest commits from current user only
chr pick --latest

# Show latest commits from current user
chr pick --latest --show

# Pick specific number of commits
chr pick --count 10

# Pick commits from today only
chr pick --today

# Pick commits from yesterday
chr pick --yesterday

# Pick commits since a specific date
chr pick --since 2024-01-01

# Pick commits until a specific date
chr pick --until 2024-01-31

# Interactive mode
chr pick --interactive

# Disable colored output
chr pick --no-color

# Show AI-friendly usage guide
chr --llm
```

### Configuration

```bash
# View current configuration
chr config

# Set branch prefix
chr config --set-key prefix --set-value "ACME-"

# Set production branch suffix
chr config --set-key suffix_prd --set-value "-prod"

# Set homologation branch suffix  
chr config --set-key suffix_hml --set-value "-stage"

# Enable/disable colors
chr config --set-key color --set-value true

# Interactive configuration setup
chr config --setup
```

## Branch Naming Convention

`chr` expects branches to follow this naming pattern:

- **Current Branch**: Can be any branch (usually a feature branch)
- **Production Branch**: `{prefix}{card-number}{suffix_prd}`
- **Homologation Branch**: `{prefix}{card-number}{suffix_hml}`

### Example

With default configuration:
- Prefix: `ZUP-`
- Production suffix: `-prd`
- Homologation suffix: `-hml`

For card number `123`:
- Production branch: `ZUP-123-prd`
- Homologation branch: `ZUP-123-hml`

## How It Works

1. **Branch Detection**: Extracts the card number from the current branch name
2. **Commit Retrieval**: Gets commits from the PRD branch that aren't in the HML branch
3. **User Filtering**: Filters to show only commits by the current Git user
4. **Date Filtering**: Applies any date-based filters specified
5. **Duplicate Detection**: Uses composite matching to identify already-picked commits (solving the rebase problem)
6. **Display/Action**: Shows commits or cherry-picks them based on the mode

## Rebase Problem Solution

Traditional tools fail when commits are rebased because Git assigns new hashes to rebased commits. `chr` solves this by using "commit signatures" that combine:

- **Author name**
- **Commit date** 
- **First line of commit message**

This allows matching commits even after their hashes change due to rebasing.

## Configuration File

Configuration is stored at `~/.config/chr.toml`:

```toml
# Configuration file for chr tool

# The prefix for branch names (default: "ZUP-")
prefix = "ZUP-"

# The suffix for production branches (default: "-prd")
suffix_prd = "-prd"

# The suffix for homologation branches (default: "-hml")
suffix_hml = "-hml"

# Enable colored output (default: true)
color = true
```

## Environment Variables

Configuration can also be set via environment variables:

- `CHR_PREFIX`: Branch name prefix
- `CHR_SUFFIX_PRD`: Production branch suffix
- `CHR_SUFFIX_HML`: Homologation branch suffix  
- `CHR_COLOR`: Enable/disable colors (`true`/`false`)

Command-line flags override environment variables, which override the config file.

## Examples

### Daily Workflow

```bash
# Check what commits need to be picked today
chr pick --today --show

# Pick all your commits from today
chr pick --today

# Check your latest commits from the last few days
chr pick --latest --show --since 2024-01-20
```

### After Rebase

```bash
# Even after rebasing, chr will still find your commits
git rebase main
chr pick --latest  # Will correctly identify and pick your rebased commits
```

### Custom Configuration

```bash
# Set up for a different project
chr config --set-key prefix --set-value "PROJ-"
chr config --set-key suffix_prd --set-value "-production"
chr config --set-key suffix_hml --set-value "-staging"

# Now chr will look for branches like PROJ-123-production and PROJ-123-staging
```

## AI-Friendly Documentation

`chr` includes comprehensive AI-friendly documentation accessible via the `--llm` flag:

```bash
# Show main LLM usage guide
chr --llm

# Show configuration-specific guide
chr config --llm
```

This provides LLMs and AI assistants with:
- Detailed usage patterns and best practices
- Common workflow examples
- Configuration management guidance
- Error handling scenarios
- Command priority recommendations

## Error Handling

`chr` provides helpful error messages for common issues:

- **Invalid branch name**: Branch doesn't match expected format
- **Missing branches**: PRD or HML branches don't exist
- **Not in Git repo**: Current directory isn't a Git repository
- **No commits found**: No commits match the criteria
- **Cherry-pick conflicts**: Git conflicts during cherry-pick operation

## Development

### Building from Source

```bash
go build -o chr .
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
chr/
├── cmd/                    # CLI commands and parsing
├── internal/
│   ├── config/            # Configuration management  
│   ├── git/               # Git operations
│   └── picker/            # Commit matching and filtering
├── testdata/              # Test fixtures
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run tests: `go test ./...`
6. Submit a pull request

## License

MIT License - see LICENSE file for details.