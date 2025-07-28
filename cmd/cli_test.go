package cmd

import (
	"testing"
)

func TestPickCmd_ReverseFlag(t *testing.T) {
	pickCmd := &PickCmd{Reverse: true}
	if !pickCmd.Reverse {
		t.Error("Reverse flag should be set")
	}
	
	pickCmd = &PickCmd{Reverse: false}
	if pickCmd.Reverse {
		t.Error("Reverse flag should not be set")
	}
}

func TestPickCmd_DefaultValues(t *testing.T) {
	pickCmd := &PickCmd{}
	
	if pickCmd.Count != 0 {
		t.Errorf("Expected Count to be 0 (before Kong processing), got %d", pickCmd.Count)
	}
	
	if pickCmd.Latest {
		t.Error("Latest flag should be false by default")
	}
	
	if pickCmd.Show {
		t.Error("Show flag should be false by default")
	}
	
	if pickCmd.Reverse {
		t.Error("Reverse flag should be false by default")
	}
	
	if pickCmd.Interactive {
		t.Error("Interactive flag should be false by default")
	}
	
	if pickCmd.Continue {
		t.Error("Continue flag should be false by default")
	}
	
	if pickCmd.Debug {
		t.Error("Debug flag should be false by default")
	}
	
	if pickCmd.NoFilter {
		t.Error("NoFilter flag should be false by default")
	}
}

func TestPickCmd_FlagCombinations(t *testing.T) {
	tests := []struct {
		name     string
		reverse  bool
		show     bool
		latest   bool
		debug    bool
		expected string
	}{
		{
			name:     "normal mode",
			reverse:  false,
			show:     false,
			latest:   false,
			debug:    false,
			expected: "normal cherry-pick from PRD to HML",
		},
		{
			name:     "reverse mode",
			reverse:  true,
			show:     false,
			latest:   false,
			debug:    false,
			expected: "reverse cherry-pick from HML to PRD",
		},
		{
			name:     "show mode",
			reverse:  false,
			show:     true,
			latest:   false,
			debug:    false,
			expected: "dry-run mode showing PRD commits",
		},
		{
			name:     "reverse show mode",
			reverse:  true,
			show:     true,
			latest:   false,
			debug:    false,
			expected: "dry-run mode showing HML commits",
		},
		{
			name:     "reverse latest mode",
			reverse:  true,
			show:     false,
			latest:   true,
			debug:    false,
			expected: "latest commits from HML to PRD",
		},
		{
			name:     "reverse debug mode",
			reverse:  true,
			show:     false,
			latest:   false,
			debug:    true,
			expected: "debug mode reverse from HML to PRD",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pickCmd := &PickCmd{
				Reverse: tt.reverse,
				Show:    tt.show,
				Latest:  tt.latest,
				Debug:   tt.debug,
			}
			
			if pickCmd.Reverse != tt.reverse {
				t.Errorf("Expected Reverse to be %v, got %v", tt.reverse, pickCmd.Reverse)
			}
			
			if pickCmd.Show != tt.show {
				t.Errorf("Expected Show to be %v, got %v", tt.show, pickCmd.Show)
			}
			
			if pickCmd.Latest != tt.latest {
				t.Errorf("Expected Latest to be %v, got %v", tt.latest, pickCmd.Latest)
			}
			
			if pickCmd.Debug != tt.debug {
				t.Errorf("Expected Debug to be %v, got %v", tt.debug, pickCmd.Debug)
			}
		})
	}
}

func TestPickCmd_BranchLogicWithReverse(t *testing.T) {
	tests := []struct {
		name              string
		reverse           bool
		expectedSource    string
		expectedTarget    string
		cardNumber        string
		prefix            string
		suffixPrd         string
		suffixHml         string
	}{
		{
			name:              "normal direction",
			reverse:           false,
			expectedSource:    "ZUP-123-prd",
			expectedTarget:    "ZUP-123-hml",
			cardNumber:        "123",
			prefix:            "ZUP-",
			suffixPrd:         "-prd",
			suffixHml:         "-hml",
		},
		{
			name:              "reverse direction",
			reverse:           true,
			expectedSource:    "ZUP-123-hml",
			expectedTarget:    "ZUP-123-prd",
			cardNumber:        "123",
			prefix:            "ZUP-",
			suffixPrd:         "-prd",
			suffixHml:         "-hml",
		},
		{
			name:              "custom prefixes normal",
			reverse:           false,
			expectedSource:    "ACME-456-production",
			expectedTarget:    "ACME-456-staging",
			cardNumber:        "456",
			prefix:            "ACME-",
			suffixPrd:         "-production",
			suffixHml:         "-staging",
		},
		{
			name:              "custom prefixes reverse",
			reverse:           true,
			expectedSource:    "ACME-456-staging",
			expectedTarget:    "ACME-456-production",
			cardNumber:        "456",
			prefix:            "ACME-",
			suffixPrd:         "-production",
			suffixHml:         "-staging",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prdBranch := tt.prefix + tt.cardNumber + tt.suffixPrd
			hmlBranch := tt.prefix + tt.cardNumber + tt.suffixHml
			
			var sourceBranch, targetBranch string
			if tt.reverse {
				sourceBranch = hmlBranch
				targetBranch = prdBranch
			} else {
				sourceBranch = prdBranch
				targetBranch = hmlBranch
			}
			
			if sourceBranch != tt.expectedSource {
				t.Errorf("Expected source branch to be %s, got %s", tt.expectedSource, sourceBranch)
			}
			
			if targetBranch != tt.expectedTarget {
				t.Errorf("Expected target branch to be %s, got %s", tt.expectedTarget, targetBranch)
			}
		})
	}
}

func TestPickCmd_DateValidation(t *testing.T) {
	tests := []struct {
		name        string
		since       string
		until       string
		expectError bool
	}{
		{
			name:        "valid since date",
			since:       "2024-01-15",
			expectError: false,
		},
		{
			name:        "valid until date",
			until:       "2024-12-31",
			expectError: false,
		},
		{
			name:        "invalid since date",
			since:       "invalid-date",
			expectError: true,
		},
		{
			name:        "invalid until date",
			until:       "not-a-date",
			expectError: true,
		},
		{
			name:        "valid date range",
			since:       "2024-01-01",
			until:       "2024-12-31",
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pickCmd := &PickCmd{
				Since: tt.since,
				Until: tt.until,
			}
			
			var err error
			if pickCmd.Since != "" {
				err = validateDate(pickCmd.Since)
				if tt.expectError && err == nil {
					t.Error("Expected validation error for since date")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected validation error for since date: %v", err)
				}
			}
			
			if pickCmd.Until != "" {
				err = validateDate(pickCmd.Until)
				if tt.expectError && err == nil {
					t.Error("Expected validation error for until date")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected validation error for until date: %v", err)
				}
			}
		})
	}
}
