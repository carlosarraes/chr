# chr

A simple CLI tool to manage Git branches and commits.

## Installation

```bash
curl -sSf https://raw.githubusercontent.com/carlosarraes/chr/main/install.sh | sh
```

This will download and install the latest release of chr to /usr/local/bin.

Or manually download the latest binary from the [releases page](https://github.com/carlosarraes/chr/releases).

## Commands

### `chr pick`

Show and cherry-pick commits from PRD branches that are not in HML branches.

#### Usage

```
chr pick [OPTIONS]
```

#### Options

- `-c, --count <COUNT>`  
  Number of commits to pick [default: 5]

- `-l, --latest`  
  Pick latest commits from the current user only (up to 100 commits)  
  *Note: Rebases might give you already picked commits*

- `-s, --show`  
  Show commits instead of picking (dry run)

#### Examples

Show the last 5 commits from PRD branch not in HML branch:
```
chr pick --show
```

Cherry-pick your latest commits (up to 100):
```
chr pick --latest
```

Cherry-pick your latest 10 commits:
```
chr pick --latest --count 10
```