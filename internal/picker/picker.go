package picker

import (
	"strings"

	"github.com/carlosarraes/chr/internal/git"
)

// CommitMatch represents a match between commits in different branches
type CommitMatch struct {
	Source git.Commit // Commit from source branch (PRD)
	Target git.Commit // Matching commit from target branch (HML)
	Score  int        // Match confidence score (0-100)
}

// CommitMatcher provides commit matching functionality
type CommitMatcher struct {
	// Future: could add configuration for matching strategies
}

// NewCommitMatcher creates a new commit matcher
func NewCommitMatcher() *CommitMatcher {
	return &CommitMatcher{}
}

// FindMatches finds matching commits between source and target lists
// This is the core function that solves the rebase hash-change problem
func (cm *CommitMatcher) FindMatches(sourceCommits, targetCommits []git.Commit) []CommitMatch {
	var matches []CommitMatch

	for _, sourceCommit := range sourceCommits {
		for _, targetCommit := range targetCommits {
			if match, score := cm.matchCommits(sourceCommit, targetCommit); match {
				matches = append(matches, CommitMatch{
					Source: sourceCommit,
					Target: targetCommit,
					Score:  score,
				})
				break // Only find first match for each source commit
			}
		}
	}

	return matches
}

// GetUnmatched returns commits from source that don't have matches in target
func (cm *CommitMatcher) GetUnmatched(sourceCommits, targetCommits []git.Commit) []git.Commit {
	matches := cm.FindMatches(sourceCommits, targetCommits)
	matchedHashes := make(map[string]bool)

	for _, match := range matches {
		matchedHashes[match.Source.Hash] = true
	}

	var unmatched []git.Commit
	for _, commit := range sourceCommits {
		if !matchedHashes[commit.Hash] {
			unmatched = append(unmatched, commit)
		}
	}

	return unmatched
}

// matchCommits determines if two commits match using various strategies
func (cm *CommitMatcher) matchCommits(source, target git.Commit) (bool, int) {
	// Strategy 1: Exact signature match (author + date + message)
	// This handles rebases where content is identical
	if source.Signature() == target.Signature() {
		return true, 100
	}

	// Strategy 2: Message + Author match (ignoring date)
	// This handles cases where commits are cherry-picked on different dates
	if cm.sameAuthorAndMessage(source, target) {
		return true, 80
	}

	// Strategy 3: Fuzzy message matching (future enhancement)
	// Could implement Levenshtein distance or other algorithms

	return false, 0
}

// sameAuthorAndMessage checks if two commits have the same author and message
func (cm *CommitMatcher) sameAuthorAndMessage(c1, c2 git.Commit) bool {
	if c1.Author != c2.Author {
		return false
	}

	// Compare first line of commit messages (ignore multiline descriptions)
	msg1 := strings.Split(c1.Message, "\n")[0]
	msg2 := strings.Split(c2.Message, "\n")[0]

	return strings.TrimSpace(msg1) == strings.TrimSpace(msg2)
}

// FilterUnpickedCommits returns commits from PRD that haven't been picked to HML
// This is the main function used by the CLI to find commits to cherry-pick
func FilterUnpickedCommits(prdCommits, hmlCommits []git.Commit) []git.Commit {
	if len(hmlCommits) == 0 {
		// If HML is empty, all PRD commits are unpicked
		return prdCommits
	}

	matcher := NewCommitMatcher()
	return matcher.GetUnmatched(prdCommits, hmlCommits)
}

// CommitGroup represents a group of related commits
type CommitGroup struct {
	Title   string
	Commits []git.Commit
}

// GroupCommitsByMessage groups commits by their message prefix (e.g., "feat:", "fix:")
func GroupCommitsByMessage(commits []git.Commit) []CommitGroup {
	groups := make(map[string][]git.Commit)
	var order []string

	for _, commit := range commits {
		prefix := extractMessagePrefix(commit.Message)
		if _, exists := groups[prefix]; !exists {
			order = append(order, prefix)
		}
		groups[prefix] = append(groups[prefix], commit)
	}

	var result []CommitGroup
	for _, prefix := range order {
		result = append(result, CommitGroup{
			Title:   prefix,
			Commits: groups[prefix],
		})
	}

	return result
}

// extractMessagePrefix extracts the conventional commit prefix (feat:, fix:, etc.)
func extractMessagePrefix(message string) string {
	firstLine := strings.Split(message, "\n")[0]
	if strings.Contains(firstLine, ":") {
		parts := strings.SplitN(firstLine, ":", 2)
		return strings.TrimSpace(parts[0]) + ":"
	}
	return "other:"
}

// CommitSummary provides a summary of commits for display
type CommitSummary struct {
	Total    int
	ByAuthor map[string]int
	ByType   map[string]int
}

// SummarizeCommits creates a summary of the provided commits
func SummarizeCommits(commits []git.Commit) CommitSummary {
	summary := CommitSummary{
		Total:    len(commits),
		ByAuthor: make(map[string]int),
		ByType:   make(map[string]int),
	}

	for _, commit := range commits {
		summary.ByAuthor[commit.Author]++

		prefix := extractMessagePrefix(commit.Message)
		summary.ByType[prefix]++
	}

	return summary
}
