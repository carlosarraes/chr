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
	Version   bool `kong:"short='v',help='Show version information'"`
	NoColor   bool `kong:"help='Disable colored output'"`
	
	// Commands
	Show   ShowCmd   `kong:"cmd,default='1',help='Show/pick commits (default command)'"`
	Config ConfigCmd `kong:"cmd,help='Manage configuration'"`
}

// ShowCmd represents the main command for showing/picking commits
type ShowCmd struct {
	Count       int    `kong:"short='c',default='5',help='Number of commits to show'"`
	Pick        bool   `kong:"help='Actually cherry-pick commits (default is dry-run)'"`
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

// Run executes the main show command
func (s *ShowCmd) Run(ctx *kong.Context, globals *CLI) error {
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
	
	// Get commits from PRD branch
	prdCommits, err := git.GetCommits(repoDir, hmlBranch, prdBranch, s.Count)
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
	
	// Filter by current user
	userCommits := git.FilterCommitsByAuthor(prdCommits, currentUser)
	
	// Apply date filtering
	var filteredCommits []git.Commit
	if s.Today {
		filteredCommits = git.FilterCommitsByDate(userCommits, git.NewTodayFilter())
	} else if s.Yesterday {
		filteredCommits = git.FilterCommitsByDate(userCommits, git.NewYesterdayFilter())
	} else if s.Since != "" {
		if err := validateDate(s.Since); err != nil {
			return fmt.Errorf("invalid since date: %w", err)
		}
		since, _ := time.Parse("2006-01-02", s.Since)
		filter := &git.DateFilter{Type: git.DateFilterTypeSince, Since: since}
		filteredCommits = git.FilterCommitsByDate(userCommits, filter)
	} else if s.Until != "" {
		if err := validateDate(s.Until); err != nil {
			return fmt.Errorf("invalid until date: %w", err)
		}
		until, _ := time.Parse("2006-01-02", s.Until)
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
	
	if !s.Pick {
		fmt.Println("\nDry-run mode. Use --pick to actually cherry-pick these commits.")
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

// validateDate validates a date string in YYYY-MM-DD format
func validateDate(dateStr string) error {
	_, err := time.Parse("2006-01-02", dateStr)
	return err
}

// beforeInterceptor handles global setup
func (cli *CLI) BeforeApply(ctx *kong.Context) error {
	// Handle version flag
	if cli.Version {
		fmt.Println("chr version 0.0.2")
		os.Exit(0)
	}
	
	return nil
}

// ExecuteCLI runs the CLI application
func ExecuteCLI(args []string) error {
	var cli CLI
	
	parser, err := kong.New(&cli,
		kong.Name("chr"),
		kong.Description("A simple CLI tool to manage Git branches and commits"),
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