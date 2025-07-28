# chr

Smart Git cherry-picking between production and homologation branches with rebase-safe commit matching.

## Quick Start

```bash
# Install
curl -sSf https://raw.githubusercontent.com/carlosarraes/chr/main/install.sh | sh

# Cherry-pick from PRD to HML (default)
chr pick

# Cherry-pick from HML to PRD  
chr pick --reverse

# Show what would be picked (dry-run)
chr pick --show
```

## Key Features

- **Bidirectional**: PRD→HML (default) or HML→PRD (`--reverse`)
- **Rebase-safe**: Matches commits by author+date+message, not hash
- **Smart filtering**: Avoids duplicate picks across rebases
- **User-focused**: `--latest` shows only your commits
- **Date filtering**: `--today`, `--yesterday`, custom ranges

## Installation

### Quick Install
```bash
curl -sSf https://raw.githubusercontent.com/carlosarraes/chr/main/install.sh | sh
```

### Manual
Download from [releases](https://github.com/carlosarraes/chr/releases) or build from source:
```bash
git clone https://github.com/carlosarraes/chr && cd chr && make install-local
```

## Branch Convention

chr expects this naming pattern:
- **Current branch**: Any branch (e.g., `feature/login`)  
- **Production**: `{prefix}{card-number}{suffix_prd}` (e.g., `ZUP-123-prd`)
- **Homologation**: `{prefix}{card-number}{suffix_hml}` (e.g., `ZUP-123-hml`)

## Configuration

### Quick Setup
```bash
chr config --set-key prefix --set-value "PROJ-"
chr config --set-key suffix_prd --set-value "-prod"  
chr config --set-key suffix_hml --set-value "-stage"
```

### Config Sources (priority order)
1. CLI flags: `chr pick --reverse`
2. Environment: `CHR_PREFIX="PROJ-" chr pick`
3. Config file: `~/.config/chr.toml`
4. Defaults: `ZUP-`, `-prd`, `-hml`

### Config File (`~/.config/chr.toml`)
```toml
prefix = "ZUP-"
suffix_prd = "-prd"
suffix_hml = "-hml"
color = true
```

## Common Workflows

### Daily Picking
```bash
# Check today's commits to pick
chr pick --today --show

# Pick them
chr pick --today

# Reverse direction (HML → PRD)
chr pick --reverse --today
```

### After Conflicts
```bash
# Resolve conflicts, then:
chr pick --continue
```

### Debugging Issues
```bash
# See what's happening
chr pick --debug --show

# Skip smart filtering
chr pick --no-filter --show
```

## Advanced Usage

| Flag | Description |
|------|-------------|
| `--reverse` | Pick HML → PRD instead of PRD → HML |
| `--show` | Dry-run mode (safe preview) |
| `--latest` | Only your commits |
| `--today`, `--yesterday` | Date filters |
| `--since DATE`, `--until DATE` | Custom date range |
| `--count N` | Limit number of commits |
| `--continue` | Resume after conflicts |
| `--debug` | Detailed output |
| `--no-filter` | Disable smart deduplication |

## How It Works

1. **Detects** card number from current branch
2. **Determines** source/target branches (respects `--reverse`)
3. **Finds** commits in source not in target
4. **Matches** commits safely using author+date+message (survives rebases)
5. **Filters** by user/date if requested
6. **Shows** or **picks** commits

## Troubleshooting

### Common Issues
- **"Branch doesn't match format"**: Check branch naming convention
- **"No commits found"**: Try `--debug --show` to see what's happening
- **"Branch doesn't exist"**: Ensure both PRD and HML branches exist
- **Cherry-pick conflicts**: Resolve manually, then `chr pick --continue`

### Debug Commands
```bash
chr pick --debug --show           # See detailed process
chr pick --no-filter --show       # Skip smart filtering  
chr config                        # Check configuration
```

## Development

```bash
make build        # Build binary
make test         # Run tests
make fmt          # Format code
make lint         # Lint code
```

## License

MIT License