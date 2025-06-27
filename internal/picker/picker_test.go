package picker

import (
	"testing"

	"github.com/carlosarraes/chr/internal/git"
)

func TestCommitMatcher_FindMatches(t *testing.T) {
	// Source commits (from PRD branch)
	sourceCommits := []git.Commit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug in login", Date: "2024-01-02"},
		{Hash: "ghi789", Author: "Test User", Message: "refactor: improve performance", Date: "2024-01-03"},
	}
	
	// Target commits (from HML branch) - same commits but different hashes due to rebase
	targetCommits := []git.Commit{
		{Hash: "xyz111", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"},
		{Hash: "xyz222", Author: "Other User", Message: "chore: update dependencies", Date: "2024-01-02"},
	}
	
	matcher := NewCommitMatcher()
	matches := matcher.FindMatches(sourceCommits, targetCommits)
	
	// Should find 1 match (the "feat: add new feature" commit)
	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}
	
	// The match should be the first source commit
	if len(matches) > 0 {
		match := matches[0]
		if match.Source.Hash != "abc123" {
			t.Errorf("Expected source hash abc123, got %s", match.Source.Hash)
		}
		if match.Target.Hash != "xyz111" {
			t.Errorf("Expected target hash xyz111, got %s", match.Target.Hash)
		}
	}
}

func TestCommitMatcher_GetUnmatched(t *testing.T) {
	sourceCommits := []git.Commit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug in login", Date: "2024-01-02"},
		{Hash: "ghi789", Author: "Test User", Message: "refactor: improve performance", Date: "2024-01-03"},
	}
	
	targetCommits := []git.Commit{
		{Hash: "xyz111", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"}, // This matches
	}
	
	matcher := NewCommitMatcher()
	unmatched := matcher.GetUnmatched(sourceCommits, targetCommits)
	
	// Should return 2 unmatched commits
	if len(unmatched) != 2 {
		t.Errorf("Expected 2 unmatched commits, got %d", len(unmatched))
	}
	
	// Check that the correct commits are unmatched
	expectedHashes := map[string]bool{"def456": true, "ghi789": true}
	for _, commit := range unmatched {
		if !expectedHashes[commit.Hash] {
			t.Errorf("Unexpected unmatched commit: %s", commit.Hash)
		}
	}
}

func TestSignatureMatching(t *testing.T) {
	commit1 := git.Commit{
		Hash:    "abc123",
		Author:  "Test User",
		Message: "feat: add new feature\n\nDetailed description here",
		Date:    "2024-01-01",
	}
	
	commit2 := git.Commit{
		Hash:    "xyz999", // Different hash (after rebase)
		Author:  "Test User",
		Message: "feat: add new feature", // Same message (first line)
		Date:    "2024-01-01",
	}
	
	commit3 := git.Commit{
		Hash:    "def456",
		Author:  "Other User", // Different author
		Message: "feat: add new feature",
		Date:    "2024-01-01",
	}
	
	if commit1.Signature() != commit2.Signature() {
		t.Errorf("Expected commits with same content to have same signature")
		t.Logf("Commit1 signature: %s", commit1.Signature())
		t.Logf("Commit2 signature: %s", commit2.Signature())
	}
	
	if commit1.Signature() == commit3.Signature() {
		t.Errorf("Expected commits with different authors to have different signatures")
	}
}

func TestCommitMatcher_MatchingStrategies(t *testing.T) {
	tests := []struct {
		name           string
		source         git.Commit
		target         git.Commit
		shouldMatch    bool
		strategy       string
	}{
		{
			name: "exact signature match",
			source: git.Commit{
				Hash: "abc123", Author: "Test User", 
				Message: "feat: add feature", Date: "2024-01-01",
			},
			target: git.Commit{
				Hash: "xyz999", Author: "Test User", 
				Message: "feat: add feature", Date: "2024-01-01",
			},
			shouldMatch: true,
			strategy:    "signature",
		},
		{
			name: "message-only match (same author, different date)",
			source: git.Commit{
				Hash: "abc123", Author: "Test User", 
				Message: "fix: resolve issue", Date: "2024-01-01",
			},
			target: git.Commit{
				Hash: "xyz999", Author: "Test User", 
				Message: "fix: resolve issue", Date: "2024-01-02",
			},
			shouldMatch: true,
			strategy:    "message",
		},
		{
			name: "no match - different author and message",
			source: git.Commit{
				Hash: "abc123", Author: "Test User", 
				Message: "feat: add feature", Date: "2024-01-01",
			},
			target: git.Commit{
				Hash: "xyz999", Author: "Other User", 
				Message: "fix: different change", Date: "2024-01-01",
			},
			shouldMatch: false,
			strategy:    "none",
		},
	}
	
	matcher := NewCommitMatcher()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCommits := []git.Commit{tt.source}
			targetCommits := []git.Commit{tt.target}
			
			matches := matcher.FindMatches(sourceCommits, targetCommits)
			
			if tt.shouldMatch {
				if len(matches) != 1 {
					t.Errorf("Expected 1 match for %s, got %d", tt.name, len(matches))
				}
			} else {
				if len(matches) != 0 {
					t.Errorf("Expected 0 matches for %s, got %d", tt.name, len(matches))
				}
			}
		})
	}
}

func TestFilterUnpickedCommits(t *testing.T) {
	prdCommits := []git.Commit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug", Date: "2024-01-02"},
		{Hash: "ghi789", Author: "Test User", Message: "docs: update readme", Date: "2024-01-03"},
	}
	
	hmlCommits := []git.Commit{
		{Hash: "xyz111", Author: "Test User", Message: "feat: add feature", Date: "2024-01-01"}, // Already picked
		{Hash: "xyz222", Author: "Other User", Message: "chore: unrelated", Date: "2024-01-04"},
	}
	
	unpicked := FilterUnpickedCommits(prdCommits, hmlCommits)
	
	// Should return 2 commits that haven't been picked yet
	if len(unpicked) != 2 {
		t.Errorf("Expected 2 unpicked commits, got %d", len(unpicked))
	}
	
	// Check that the correct commits are returned
	expectedMessages := map[string]bool{
		"fix: resolve bug": true,
		"docs: update readme": true,
	}
	
	for _, commit := range unpicked {
		if !expectedMessages[commit.Message] {
			t.Errorf("Unexpected unpicked commit: %s", commit.Message)
		}
	}
}

func TestFilterUnpickedCommits_EmptyHML(t *testing.T) {
	prdCommits := []git.Commit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug", Date: "2024-01-02"},
	}
	
	var hmlCommits []git.Commit // Empty HML branch
	
	unpicked := FilterUnpickedCommits(prdCommits, hmlCommits)
	
	// Should return all PRD commits since HML is empty
	if len(unpicked) != 2 {
		t.Errorf("Expected 2 unpicked commits, got %d", len(unpicked))
	}
}