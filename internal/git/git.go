package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	Hash    string
	Author  string
	Message string
	Date    string
}

type DateFilterType int

const (
	DateFilterTypeToday DateFilterType = iota
	DateFilterTypeYesterday
	DateFilterTypeSince
	DateFilterTypeUntil
	DateFilterTypeRange
)

type DateFilter struct {
	Type  DateFilterType
	Since time.Time
	Until time.Time
}

// NewTodayFilter creates a filter for today's commits
func NewTodayFilter() *DateFilter {
	return &DateFilter{
		Type:  DateFilterTypeToday,
		Since: time.Now().Truncate(24 * time.Hour),
		Until: time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
	}
}

// NewYesterdayFilter creates a filter for yesterday's commits
func NewYesterdayFilter() *DateFilter {
	return &DateFilter{
		Type:  DateFilterTypeYesterday,
		Since: time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour),
		Until: time.Now().Truncate(24 * time.Hour),
	}
}

// GetCurrentBranch returns the current git branch name
func GetCurrentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCurrentUser returns the current git user name
func GetCurrentUser(repoDir string) (string, error) {
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git user name: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// BranchExists checks if a git branch exists
func BranchExists(repoDir, branch string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = repoDir
	err := cmd.Run()
	if err != nil {
		// If the command fails, the branch doesn't exist
		return false, nil
	}
	return true, nil
}

func FetchBranches(repoDir string, debug bool, branches ...string) error {
	checkCmd := exec.Command("git", "remote", "get-url", "origin")
	checkCmd.Dir = repoDir
	if err := checkCmd.Run(); err != nil {
		if debug {
			fmt.Printf("Debug: No origin remote found, skipping fetch\n")
		}
		return nil
	}

	if debug {
		fmt.Printf("Debug: Fetching from origin...\n")
	}
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from origin: %w", err)
	}
	if debug {
		fmt.Printf("Debug: Fetch completed successfully\n")
	}
	return nil
}

// GetCommits gets commits that are in sourceBranch but not in targetBranch
func GetCommits(repoDir, targetBranch, sourceBranch string, limit int, debug bool) ([]Commit, error) {
	var sourceRef, targetRef string
	
	if localExists, _ := BranchExists(repoDir, sourceBranch); localExists {
		sourceRef = sourceBranch
		if debug {
			fmt.Printf("Debug: Using local source branch: %s\n", sourceRef)
		}
	} else {
		if err := FetchBranches(repoDir, debug, targetBranch, sourceBranch); err != nil {
			return nil, fmt.Errorf("failed to fetch branches: %w", err)
		}
		
		sourceRemoteRef := fmt.Sprintf("origin/%s", sourceBranch)
		if remoteExists, _ := BranchExists(repoDir, sourceRemoteRef); remoteExists {
			sourceRef = sourceRemoteRef
			if debug {
				fmt.Printf("Debug: Local source branch not found, using remote: %s\n", sourceRef)
			}
		} else {
			sourceRef = sourceBranch
			if debug {
				fmt.Printf("Debug: Neither local nor remote source branch found, using: %s\n", sourceRef)
			}
		}
	}
	
	if localExists, _ := BranchExists(repoDir, targetBranch); localExists {
		targetRef = targetBranch
		if debug {
			fmt.Printf("Debug: Using local target branch: %s\n", targetRef)
		}
	} else {
		if err := FetchBranches(repoDir, debug, targetBranch, sourceBranch); err != nil {
			return nil, fmt.Errorf("failed to fetch branches: %w", err)
		}
		
		targetRemoteRef := fmt.Sprintf("origin/%s", targetBranch)
		if remoteExists, _ := BranchExists(repoDir, targetRemoteRef); remoteExists {
			targetRef = targetRemoteRef
			if debug {
				fmt.Printf("Debug: Local target branch not found, using remote: %s\n", targetRef)
			}
		} else {
			targetRef = targetBranch
			if debug {
				fmt.Printf("Debug: Neither local nor remote target branch found, using: %s\n", targetRef)
			}
		}
	}

	// Use git log to find commits in sourceBranch that are not in targetBranch
	args := []string{"log",
		fmt.Sprintf("^%s", targetRef),
		sourceRef,
		"--format=%h|%an|%s|%ad",
		"--date=short",
	}

	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	if debug {
		fmt.Printf("Debug: Running git command: git %s\n", strings.Join(args, " "))
		fmt.Printf("Debug: sourceRef=%s, targetRef=%s\n", sourceRef, targetRef)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		if debug {
			if exitError, ok := err.(*exec.ExitError); ok {
				fmt.Printf("Debug: Git command failed with stderr: %s\n", string(exitError.Stderr))
			}
		}
		return nil, fmt.Errorf("failed to get commits with command 'git %s': %w", strings.Join(args, " "), err)
	}
	
	if debug {
		fmt.Printf("Debug: Git command succeeded, output length: %d bytes\n", len(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []Commit{}, nil // No commits found
	}

	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		commit := Commit{
			Hash:    parts[0],
			Author:  parts[1],
			Message: parts[2],
			Date:    parts[3],
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// FilterCommitsByAuthor filters commits by author name
func FilterCommitsByAuthor(commits []Commit, author string) []Commit {
	filtered := make([]Commit, 0)
	for _, commit := range commits {
		if commit.Author == author {
			filtered = append(filtered, commit)
		}
	}
	return filtered
}

// FilterCommitsByDate filters commits by date criteria
func FilterCommitsByDate(commits []Commit, filter *DateFilter) []Commit {
	if filter == nil {
		return commits
	}

	filtered := make([]Commit, 0)
	for _, commit := range commits {
		commitDate, err := time.Parse("2006-01-02", commit.Date)
		if err != nil {
			continue // Skip commits with invalid dates
		}

		switch filter.Type {
		case DateFilterTypeToday:
			today := time.Now().Truncate(24 * time.Hour)
			if commitDate.Equal(today) || (commitDate.After(today) && commitDate.Before(today.Add(24*time.Hour))) {
				filtered = append(filtered, commit)
			}
		case DateFilterTypeYesterday:
			yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
			if commitDate.Equal(yesterday) || (commitDate.After(yesterday) && commitDate.Before(yesterday.Add(24*time.Hour))) {
				filtered = append(filtered, commit)
			}
		case DateFilterTypeSince:
			if commitDate.Equal(filter.Since) || commitDate.After(filter.Since) {
				filtered = append(filtered, commit)
			}
		case DateFilterTypeUntil:
			if commitDate.Equal(filter.Until) || commitDate.Before(filter.Until) {
				filtered = append(filtered, commit)
			}
		case DateFilterTypeRange:
			if (commitDate.Equal(filter.Since) || commitDate.After(filter.Since)) &&
				(commitDate.Equal(filter.Until) || commitDate.Before(filter.Until)) {
				filtered = append(filtered, commit)
			}
		}
	}

	return filtered
}

func CherryPickCommits(repoDir string, commitHashes []string) error {
	if len(commitHashes) == 0 {
		return nil
	}

	fmt.Printf("Cherry-picking %d commits...\n", len(commitHashes))

	oldestCommit := commitHashes[len(commitHashes)-1]
	newestCommit := commitHashes[0]
	commitRange := fmt.Sprintf("%s^..%s", oldestCommit, newestCommit)

	revListCmd := exec.Command("git", "rev-list", "--reverse", commitRange)
	revListCmd.Dir = repoDir
	revListOutput, err := revListCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit range: %v", err)
	}

	cherryPickCmd := exec.Command("git", "cherry-pick", "--stdin")
	cherryPickCmd.Dir = repoDir
	cherryPickCmd.Stdin = strings.NewReader(string(revListOutput))

	if err := cherryPickCmd.Run(); err != nil {
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = repoDir
		statusOutput, _ := statusCmd.Output()

		fmt.Println("\nConflicts found - needs to be resolved.")

		if len(statusOutput) > 0 {
			fmt.Printf("\nFiles with conflicts:\n%s", string(statusOutput))
		}

		fmt.Println("\nWhat to do:")
		fmt.Println("1. Resolve the conflicts in the files listed above")
		fmt.Println("2. Add the resolved files: git add <file>")
		fmt.Println("3. Continue: chr pick --continue")
		fmt.Println("4. Or abort: git cherry-pick --abort")

		return nil
	}

	fmt.Println("✓ Successfully cherry-picked all commits!")
	return nil
}

// Signature creates a unique signature for a commit (for rebase-safe comparison)
func (c Commit) Signature() string {
	return fmt.Sprintf("%s:%s:%s", c.Author, c.Date, strings.Split(c.Message, "\n")[0])
}

// ParseBranchName extracts card number from branch name using the given prefix and suffix
func ParseBranchName(branchName, prefix string) (string, error) {
	if !strings.HasPrefix(branchName, prefix) {
		return "", fmt.Errorf("branch '%s' doesn't start with prefix '%s'", branchName, prefix)
	}

	// Remove prefix and extract card number (everything until first suffix or end)
	withoutPrefix := strings.TrimPrefix(branchName, prefix)
	parts := strings.Split(withoutPrefix, "-")
	if len(parts) == 0 {
		return "", fmt.Errorf("no card number found in branch name '%s'", branchName)
	}

	// First part should be the card number
	cardNumber := parts[0]
	if cardNumber == "" {
		return "", fmt.Errorf("empty card number in branch name '%s'", branchName)
	}

	// Validate that it's numeric (optional - depends on your naming convention)
	if _, err := strconv.Atoi(cardNumber); err != nil {
		// If it's not numeric, just return it as-is
		// This allows for non-numeric card numbers like "FEAT-123"
	}

	return cardNumber, nil
}
