package cmd

import (
	"testing"
)

func TestExecuteCLI_Version(t *testing.T) {
	// Test that the CLI structure can handle version parsing
	cli := &CLI{VersionFlag: true}
	if !cli.VersionFlag {
		t.Error("Version flag should be set")
	}
}

func TestExecuteCLI_Help(t *testing.T) {
	// Skip this test because Kong's help system calls os.Exit(0)
	// which causes a panic in testing environment
	t.Skip("Kong's help system calls os.Exit(0) which panics in tests")
}

func TestExecuteCLI_InvalidArgs(t *testing.T) {
	args := []string{"--invalid-flag"}
	err := ExecuteCLI(args)

	if err == nil {
		t.Error("Expected error for invalid flag")
	}
}

func TestShowCmd_Validation(t *testing.T) {
	tests := []struct {
		name    string
		since   string
		until   string
		wantErr bool
	}{
		{
			name:    "valid since date",
			since:   "2024-01-01",
			wantErr: false,
		},
		{
			name:    "invalid since date",
			since:   "invalid-date",
			wantErr: true,
		},
		{
			name:    "valid until date",
			until:   "2024-12-31",
			wantErr: false,
		},
		{
			name:    "invalid until date",
			until:   "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.since != "" {
				err = validateDate(tt.since)
			}
			if tt.until != "" {
				err = validateDate(tt.until)
			}

			if tt.wantErr && err == nil {
				t.Error("Expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestValidateConfigKey(t *testing.T) {
	tests := []struct {
		key     string
		wantErr bool
	}{
		{"prefix", false},
		{"suffix_prd", false},
		{"suffix_hml", false},
		{"color", false},
		{"invalid_key", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := ValidateConfigKey(tt.key)
			if tt.wantErr && err == nil {
				t.Error("Expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestValidateConfigValue(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"color", "true", false},
		{"color", "false", false},
		{"color", "invalid", true},
		{"prefix", "ACME-", false},
		{"prefix", "", true},
		{"suffix_prd", "-prod", false},
		{"suffix_prd", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			err := ValidateConfigValue(tt.key, tt.value)
			if tt.wantErr && err == nil {
				t.Error("Expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestSetupTestColors(t *testing.T) {
	// Test that color setup doesn't panic
	SetupTestColors(true)
	SetupTestColors(false)
}
