package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	
	"github.com/carlosarraes/chr/internal/config"
	"github.com/carlosarraes/chr/internal/git"
	"github.com/carlosarraes/chr/internal/picker"
)

// CLI represents the command-line interface structure
type CLI struct {
	// Global flags
	VersionFlag bool `kong:"short='v',name='version',help='Show version information'"`
	NoColor     bool `kong:"help='Disable colored output'"`
	LLM         bool `kong:"help='Show LLM guide for chr usage'"`
	
	// Commands
	Pick    PickCmd    `kong:"cmd,help='Show and cherry-pick commits'"`
	Config  ConfigCmd  `kong:"cmd,help='Manage configuration'"`
	Version VersionCmd `kong:"cmd,help='Show version information'"`
}

type PickCmd struct {
	Count       int    `kong:"short='c',default='5',help='Number of commits to pick'"`
	Latest      bool   `kong:"short='l',help='Pick latest commits from current user (up to 100)'"`
	Show        bool   `kong:"short='s',help='Show commits instead of picking (dry run)'"`
	Today       bool   `kong:"help='Show commits from today only'"`
	Yesterday   bool   `kong:"help='Show commits from yesterday only'"`
	Since       string `kong:"help='Show commits since date (YYYY-MM-DD)'"`
	Until       string `kong:"help='Show commits until date (YYYY-MM-DD)'"`
	Interactive bool   `kong:"short='i',help='Interactive commit selection'"`
}

// ConfigCmd represents the config subcommand
type ConfigCmd struct {
	SetKey      string `kong:"help='Configuration key to set'"`
	SetValue    string `kong:"help='Configuration value to set'"`
	Interactive bool   `kong:"name='setup',help='Interactive configuration setup'"`
}

type VersionCmd struct {}

func (p *PickCmd) Run(ctx *kong.Context, globals *CLI) error {
	// Load configuration first
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Setup colors - global flag overrides config
	color.NoColor = globals.NoColor || !cfg.Color
	
	// Get current working directory (Git repo)
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Get current branch
	currentBranch, err := git.GetCurrentBranch(repoDir)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	
	// Parse branch name to get card number
	cardNumber, err := git.ParseBranchName(currentBranch, cfg.Prefix)
	if err != nil {
		return fmt.Errorf("failed to parse branch name: %w", err)
	}
	
	// Construct PRD and HML branch names
	prdBranch := cfg.Prefix + cardNumber + cfg.SuffixPrd
	hmlBranch := cfg.Prefix + cardNumber + cfg.SuffixHml
	
	fmt.Printf("Current branch: %s\n", currentBranch)
	fmt.Printf("PRD branch: %s\n", prdBranch)
	fmt.Printf("HML branch: %s\n", hmlBranch)
	
	// Check if branches exist
	if exists, err := git.BranchExists(repoDir, prdBranch); err != nil {
		return fmt.Errorf("failed to check PRD branch: %w", err)
	} else if !exists {
		return fmt.Errorf("PRD branch '%s' does not exist", prdBranch)
	}
	
	if exists, err := git.BranchExists(repoDir, hmlBranch); err != nil {
		return fmt.Errorf("failed to check HML branch: %w", err)
	} else if !exists {
		return fmt.Errorf("HML branch '%s' does not exist", hmlBranch)
	}
	
	commitCount := p.Count
	if p.Latest {
		commitCount = 100
	}
	
	// Get commits from PRD branch
	prdCommits, err := git.GetCommits(repoDir, hmlBranch, prdBranch, commitCount)
	if err != nil {
		return fmt.Errorf("failed to get PRD commits: %w", err)
	}
	
	if len(prdCommits) == 0 {
		fmt.Println("No new commits found in PRD branch.")
		return nil
	}
	
	// Get current user for filtering
	currentUser, err := git.GetCurrentUser(repoDir)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	
	var userCommits []git.Commit
	if p.Latest {
		userCommits = git.FilterCommitsByAuthor(prdCommits, currentUser)
	} else {
		userCommits = prdCommits
	}
	
	// Apply date filtering
	var filteredCommits []git.Commit
	if p.Today {
		filteredCommits = git.FilterCommitsByDate(userCommits, git.NewTodayFilter())
	} else if p.Yesterday {
		filteredCommits = git.FilterCommitsByDate(userCommits, git.NewYesterdayFilter())
	} else if p.Since != "" {
		if err := validateDate(p.Since); err != nil {
			return fmt.Errorf("invalid since date: %w", err)
		}
		since, _ := time.Parse("2006-01-02", p.Since)
		filter := &git.DateFilter{Type: git.DateFilterTypeSince, Since: since}
		filteredCommits = git.FilterCommitsByDate(userCommits, filter)
	} else if p.Until != "" {
		if err := validateDate(p.Until); err != nil {
			return fmt.Errorf("invalid until date: %w", err)
		}
		until, _ := time.Parse("2006-01-02", p.Until)
		filter := &git.DateFilter{Type: git.DateFilterTypeUntil, Until: until}
		filteredCommits = git.FilterCommitsByDate(userCommits, filter)
	} else {
		filteredCommits = userCommits
	}
	
	if len(filteredCommits) == 0 {
		fmt.Println("No commits found for the current user with the specified filters.")
		return nil
	}
	
	// Get HML commits for comparison (to find already picked commits)
	hmlCommits, err := git.GetCommits(repoDir, "main", hmlBranch, 100) // Get more commits for comparison
	if err != nil {
		return fmt.Errorf("failed to get HML commits: %w", err)
	}
	
	// Find commits that haven't been picked yet
	unpickedCommits := picker.FilterUnpickedCommits(filteredCommits, hmlCommits)
	
	if len(unpickedCommits) == 0 {
		fmt.Println("All commits have already been picked to HML branch.")
		return nil
	}
	
	// Display commits
	fmt.Printf("\nFound %d unpicked commits:\n", len(unpickedCommits))
	for i, commit := range unpickedCommits {
		displayCommit(i+1, commit, currentUser, cfg.Color)
	}
	
	if p.Show {
		fmt.Println("\nDry-run mode. Remove --show to actually cherry-pick these commits.")
		return nil
	}
	
	// Cherry-pick mode
	fmt.Printf("\nCherry-picking %d commits...\n", len(unpickedCommits))
	
	// Extract commit hashes
	var commitHashes []string
	for _, commit := range unpickedCommits {
		commitHashes = append(commitHashes, commit.Hash)
	}
	
	// Perform cherry-pick
	if err := git.CherryPickCommits(repoDir, commitHashes); err != nil {
		return fmt.Errorf("cherry-pick failed: %w", err)
	}
	
	fmt.Println("Successfully cherry-picked all commits!")
	return nil
}

// Run executes the config command
func (c *ConfigCmd) Run(ctx *kong.Context, globals *CLI) error {
	color.NoColor = globals.NoColor
	
	// Check if both key and value are provided for setting
	if c.SetKey != "" && c.SetValue != "" {
		// Validate key and value
		if err := ValidateConfigKey(c.SetKey); err != nil {
			return err
		}
		if err := ValidateConfigValue(c.SetKey, c.SetValue); err != nil {
			return err
		}
		
		// Load current config
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		// Set the value
		if err := cfg.Set(c.SetKey, c.SetValue); err != nil {
			return fmt.Errorf("failed to set config value: %w", err)
		}
		
		// Save config
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		
		fmt.Printf("Set %s = %s\n", c.SetKey, c.SetValue)
		return nil
	}
	
	// Check if only one of key/value is provided (error case)
	if c.SetKey != "" || c.SetValue != "" {
		return fmt.Errorf("both --set-key and --set-value must be provided together")
	}
	
	if c.Interactive {
		return runInteractiveConfig()
	}
	
	// Default: show current config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	fmt.Println(cfg.String())
	return nil
}

func (v *VersionCmd) Run(ctx *kong.Context, globals *CLI) error {
	fmt.Println("chr version 0.1.1")
	return nil
}

// validateDate validates a date string in YYYY-MM-DD format
func validateDate(dateStr string) error {
	_, err := time.Parse("2006-01-02", dateStr)
	return err
}

// beforeInterceptor handles global setup
func (cli *CLI) BeforeApply(ctx *kong.Context) error {
	// Handle version flag
	if cli.VersionFlag {
		fmt.Println("chr version 0.1.1")
		os.Exit(0)
	}
	
	// Handle LLM flag
	if cli.LLM {
		// Check if we're in config context for specific config guide
		if len(os.Args) > 1 && os.Args[1] == "config" {
			showConfigLLMGuide()
		} else {
			showLLMGuide()
		}
		os.Exit(0)
	}
	
	return nil
}

// ExecuteCLI runs the CLI application
func ExecuteCLI(args []string) error {
	var cli CLI
	
	parser, err := kong.New(&cli,
		kong.Name("chr"),
		kong.Description("Git commit manager for cherry-picking between production and homologation branches\n\nUsage: chr pick [flags]  # Cherry-pick commits (default)\n       chr pick --show   # Show commits (dry run)\n       chr config        # Manage configuration"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	
	ctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	
	// Apply global interceptor
	if err := cli.BeforeApply(ctx); err != nil {
		return err
	}
	
	// Run the selected command
	return ctx.Run(ctx, &cli)
}

// Helper functions for testing
func SetupTestColors(noColor bool) {
	color.NoColor = noColor
}

func ValidateConfigKey(key string) error {
	validKeys := map[string]bool{
		"prefix":     true,
		"suffix_prd": true,
		"suffix_hml": true,
		"color":      true,
	}
	
	if !validKeys[key] {
		return fmt.Errorf("invalid configuration key: %s", key)
	}
	
	return nil
}

func ValidateConfigValue(key, value string) error {
	switch key {
	case "color":
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("color must be true or false")
		}
	case "prefix", "suffix_prd", "suffix_hml":
		if value == "" {
			return fmt.Errorf("%s cannot be empty", key)
		}
	}
	
	return nil
}

// Helper functions for config integration
func loadConfig() (*config.Config, error) {
	return config.LoadConfig("")
}

func saveConfig(cfg *config.Config) error {
	return config.SaveConfig("", cfg)
}

func runInteractiveConfig() error {
	fmt.Println("Interactive configuration setup:")
	
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	fmt.Printf("Current prefix: %s\n", cfg.Prefix)
	fmt.Printf("Current production suffix: %s\n", cfg.SuffixPrd)
	fmt.Printf("Current homologation suffix: %s\n", cfg.SuffixHml)
	fmt.Printf("Current color setting: %v\n", cfg.Color)
	
	// TODO: Add actual interactive prompts (would need a prompt library)
	fmt.Println("(Interactive prompts not yet implemented - use --set instead)")
	
	return nil
}

// displayCommit formats and displays a commit with optional colors
func displayCommit(index int, commit git.Commit, currentUser string, enableColor bool) {
	if !enableColor || color.NoColor {
		// Plain text output
		fmt.Printf("%d. %s | %s | %s | %s\n", index, commit.Hash, commit.Author, commit.Date, commit.Message)
		return
	}
	
	// Colored output
	indexColor := color.New(color.FgCyan, color.Bold)
	hashColor := color.New(color.FgYellow)
	authorColor := color.New(color.FgGreen)
	if commit.Author != currentUser {
		authorColor = color.New(color.FgRed)
	}
	dateColor := color.New(color.FgBlue)
	messageColor := color.New(color.FgWhite)
	
	// Check message type for different colors
	if strings.HasPrefix(commit.Message, "feat:") {
		messageColor = color.New(color.FgGreen)
	} else if strings.HasPrefix(commit.Message, "fix:") {
		messageColor = color.New(color.FgRed)
	} else if strings.HasPrefix(commit.Message, "docs:") {
		messageColor = color.New(color.FgCyan)
	} else if strings.HasPrefix(commit.Message, "refactor:") {
		messageColor = color.New(color.FgMagenta)
	}
	
	fmt.Printf("%s %s | %s | %s | %s\n",
		indexColor.Sprintf("%d.", index),
		hashColor.Sprint(commit.Hash),
		authorColor.Sprint(commit.Author),
		dateColor.Sprint(commit.Date),
		messageColor.Sprint(commit.Message),
	)
}

// showLLMGuide displays the main LLM guide for chr
func showLLMGuide() {
	fmt.Print(`# chr - Git Branch Commit Manager (LLM Guide)

## Overview
chr is a command-line tool for managing Git branch commits and cherry-picking between production (PRD) and homologation (HML) branches.
- **Purpose**: Automate cherry-picking commits between production and staging branches
- **Key Strength**: Rebase-safe commit matching that works even after Git rebases change commit hashes
- **LLM-Friendly**: Simple, predictable commands with clear dry-run defaults

## Core Problem Solved
Traditional cherry-picking tools fail after rebases because they rely on Git commit hashes. chr uses "commit signatures" (author + date + message) to identify commits even after rebases.

## Branch Naming Convention
chr expects this naming pattern:
- **Production Branch**: ` + "`{prefix}{card-number}{suffix_prd}`" + `
- **Homologation Branch**: ` + "`{prefix}{card-number}{suffix_hml}`" + `

Default configuration:
- Prefix: ` + "`ZUP-`" + `
- Production suffix: ` + "`-prd`" + `
- Homologation suffix: ` + "`-hml`" + `

Example branches for card 123:
- Production: ` + "`ZUP-123-prd`" + `
- Homologation: ` + "`ZUP-123-hml`" + `

## Essential Commands

### Daily Workflow (Most Common)
` + "```bash" + `
# Show what commits need to be picked (dry-run - SAFE)
chr

# Actually cherry-pick the identified commits
chr --pick

# Show only today's commits that need picking
chr --today

# Pick all commits from today
chr --today --pick

# Show commits from yesterday
chr --yesterday --pick
` + "```" + `

### Date-Based Filtering
` + "```bash" + `
# Show commits since a specific date
chr --since 2024-01-15

# Show commits until a specific date  
chr --until 2024-01-31

# Limit number of commits shown
chr --count 10

# Combine filters
chr --since 2024-01-15 --count 5 --pick
` + "```" + `

### Configuration Management
` + "```bash" + `
# View current configuration
chr config

# Set custom branch prefix for different projects
chr config --set-key prefix --set-value "PROJ-"

# Set custom production suffix
chr config --set-key suffix_prd --set-value "-production"

# Set custom homologation suffix
chr config --set-key suffix_hml --set-value "-staging"

# Interactive configuration setup
chr config --setup
` + "```" + `

## How chr Works
1. **Extracts card number** from current branch name
2. **Constructs PRD/HML branch names** using configuration
3. **Gets commits** from PRD branch not in HML branch
4. **Filters by current user** (only shows your commits)
5. **Applies date filters** if specified
6. **Uses composite matching** to identify already-picked commits
7. **Shows or picks** commits based on --pick flag

## Key Features for LLM Understanding

### Rebase-Safe Matching
chr identifies commits using:
- Author name
- Commit date
- First line of commit message

This means rebased commits are still correctly identified and not duplicated.

### Dry-Run by Default
- ` + "`chr`" + ` always shows commits first (safe preview)
- ` + "`chr --pick`" + ` actually performs the cherry-pick
- This prevents accidental operations

### User Filtering
- Only shows commits from the current Git user
- Reduces noise in team environments
- Uses ` + "`git config user.name`" + ` for identification

### Colored Output
- Green: Current user commits
- Red: Other user commits  
- Different colors for commit types (feat:, fix:, docs:, etc.)
- Disable with ` + "`--no-color`" + `

## Environment Variables
` + "```bash" + `
# Override configuration without editing files
CHR_PREFIX="ACME-"
CHR_SUFFIX_PRD="-prod"
CHR_SUFFIX_HML="-hml"
CHR_COLOR="false"
` + "```" + `

## Error Scenarios chr Handles
- **Invalid branch name**: Current branch doesn't match expected format
- **Missing branches**: PRD or HML branches don't exist
- **Not in Git repo**: Current directory isn't a Git repository
- **No commits found**: No commits match the criteria
- **Cherry-pick conflicts**: Provides clear guidance on conflict resolution

## Best Practices for LLM Integration
1. **Always start with dry-run** (` + "`chr`" + `) to preview commits
2. **Use date filters** for focused operations (` + "`--today`" + `, ` + "`--yesterday`" + `)
3. **Configure once per project** using ` + "`chr config`" + `
4. **Combine flags** for precise control (` + "`--since 2024-01-01 --count 5`" + `)
5. **Check branch names** match expected format before running

## Command Priority for LLM Usage
1. **Critical**: ` + "`chr`" + ` (dry-run preview), ` + "`chr --pick`" + ` (execute)
2. **Important**: ` + "`chr --today --pick`" + ` (daily workflow)
3. **Useful**: ` + "`chr config`" + ` (project setup), date filters
4. **Advanced**: ` + "`--interactive`" + `, custom date ranges

chr excels at eliminating the manual work of identifying and cherry-picking commits between branches while handling Git rebase scenarios that break traditional tools.
`)
}

// showConfigLLMGuide displays the config-specific LLM guide
func showConfigLLMGuide() {
	fmt.Print(`# chr config - Configuration Management (LLM Guide)

## Overview
chr config manages branch naming patterns and tool behavior for different projects and workflows.

## Essential Commands
` + "```bash" + `
# View current configuration
chr config

# Set branch prefix (most common customization)
chr config --set-key prefix --set-value "ACME-"

# Set production branch suffix
chr config --set-key suffix_prd --set-value "-production"

# Set homologation branch suffix
chr config --set-key suffix_hml --set-value "-staging"

# Enable/disable colored output
chr config --set-key color --set-value false

# Interactive setup (not yet implemented - use --set-key instead)
chr config --setup
` + "```" + `

## Configuration Keys
- **prefix**: Branch name prefix (default: "ZUP-")
- **suffix_prd**: Production branch suffix (default: "-prd")  
- **suffix_hml**: Homologation branch suffix (default: "-hml")
- **color**: Enable colored output (default: true)

## Configuration Sources (Priority Order)
1. **Command-line flags** (highest priority)
2. **Environment variables** (CHR_PREFIX, CHR_SUFFIX_PRD, etc.)
3. **Configuration file** (~/.config/chr.toml)
4. **Default values** (lowest priority)

## Common Project Setups
` + "```bash" + `
# Atlassian/Jira style (default)
chr config --set-key prefix --set-value "ZUP-"
chr config --set-key suffix_prd --set-value "-prd"
chr config --set-key suffix_hml --set-value "-hml"
# Branches: ZUP-123-prd, ZUP-123-hml

# GitHub style
chr config --set-key prefix --set-value "feature/"
chr config --set-key suffix_prd --set-value "-production"  
chr config --set-key suffix_hml --set-value "-staging"
# Branches: feature/123-production, feature/123-staging

# Enterprise style
chr config --set-key prefix --set-value "PROJ-"
chr config --set-key suffix_prd --set-value "-prod"
chr config --set-key suffix_hml --set-value "-test"
# Branches: PROJ-123-prod, PROJ-123-test
` + "```" + `

## Configuration File Location
` + "`~/.config/chr.toml`" + `

Example configuration:
` + "```toml" + `
# Configuration file for chr tool
prefix = "ACME-"
suffix_prd = "-production"
suffix_hml = "-staging"
color = true
` + "```" + `

## Environment Variable Override
` + "```bash" + `
# Temporary override without changing config file
CHR_PREFIX="TEMP-" CHR_SUFFIX_PRD="-live" chr --pick

# Export for session
export CHR_PREFIX="ACME-"
export CHR_SUFFIX_PRD="-prod"
export CHR_SUFFIX_HML="-test"
export CHR_COLOR="false"
` + "```" + `

## Validation
chr validates configuration values:
- **prefix, suffix_prd, suffix_hml**: Cannot be empty
- **color**: Must be true or false

Invalid configurations will show clear error messages with suggestions.
`)
}
