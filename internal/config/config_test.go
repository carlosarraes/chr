package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_Default(t *testing.T) {
	// Test loading default configuration when no config file exists
	// Use a non-existent path to ensure defaults are loaded
	nonExistentPath := filepath.Join(t.TempDir(), "non-existent.toml")
	cfg, err := LoadConfig(nonExistentPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Prefix != DefaultPrefix {
		t.Errorf("Expected prefix %q, got %q", DefaultPrefix, cfg.Prefix)
	}
	if cfg.SuffixPrd != DefaultSuffixPrd {
		t.Errorf("Expected suffix_prd %q, got %q", DefaultSuffixPrd, cfg.SuffixPrd)
	}
	if cfg.SuffixHml != DefaultSuffixHml {
		t.Errorf("Expected suffix_hml %q, got %q", DefaultSuffixHml, cfg.SuffixHml)
	}
	if !cfg.Color {
		t.Error("Expected color to be true by default")
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "chr.toml")
	
	configContent := `
prefix = "ACME-"
suffix_prd = "-prod"
suffix_hml = "-stage"
color = false
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Prefix != "ACME-" {
		t.Errorf("Expected prefix %q, got %q", "ACME-", cfg.Prefix)
	}
	if cfg.SuffixPrd != "-prod" {
		t.Errorf("Expected suffix_prd %q, got %q", "-prod", cfg.SuffixPrd)
	}
	if cfg.SuffixHml != "-stage" {
		t.Errorf("Expected suffix_hml %q, got %q", "-stage", cfg.SuffixHml)
	}
	if cfg.Color {
		t.Error("Expected color to be false")
	}
}

func TestLoadConfig_FromEnv(t *testing.T) {
	// Clear any existing CHR_ environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CHR_") {
			key := strings.SplitN(env, "=", 2)[0]
			t.Setenv(key, "")
		}
	}
	
	// Set environment variables
	t.Setenv("CHR_PREFIX", "ENV-")
	t.Setenv("CHR_SUFFIX_PRD", "-env-prod")
	t.Setenv("CHR_SUFFIX_HML", "-env-stage")
	t.Setenv("CHR_COLOR", "false")

	// Use non-existent config file to avoid interference
	nonExistentPath := filepath.Join(t.TempDir(), "non-existent.toml")
	cfg, err := LoadConfig(nonExistentPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Prefix != "ENV-" {
		t.Errorf("Expected prefix %q, got %q", "ENV-", cfg.Prefix)
	}
	if cfg.SuffixPrd != "-env-prod" {
		t.Errorf("Expected suffix_prd %q, got %q", "-env-prod", cfg.SuffixPrd)
	}
	if cfg.SuffixHml != "-env-stage" {
		t.Errorf("Expected suffix_hml %q, got %q", "-env-stage", cfg.SuffixHml)
	}
	if cfg.Color {
		t.Error("Expected color to be false from env")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "chr.toml")

	cfg := &Config{
		Prefix:    "TEST-",
		SuffixPrd: "-test-prod",
		SuffixHml: "-test-stage",
		Color:     false,
	}

	err := SaveConfig(configFile, cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load and verify content
	loadedCfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Prefix != cfg.Prefix {
		t.Errorf("Expected prefix %q, got %q", cfg.Prefix, loadedCfg.Prefix)
	}
	if loadedCfg.SuffixPrd != cfg.SuffixPrd {
		t.Errorf("Expected suffix_prd %q, got %q", cfg.SuffixPrd, loadedCfg.SuffixPrd)
	}
	if loadedCfg.SuffixHml != cfg.SuffixHml {
		t.Errorf("Expected suffix_hml %q, got %q", cfg.SuffixHml, loadedCfg.SuffixHml)
	}
	if loadedCfg.Color != cfg.Color {
		t.Errorf("Expected color %v, got %v", cfg.Color, loadedCfg.Color)
	}
}

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath returned empty string")
	}
	if filepath.Base(path) != "chr.toml" {
		t.Errorf("Expected config file name to be 'chr.toml', got %q", filepath.Base(path))
	}
}

func TestConfig_Set(t *testing.T) {
	cfg := &Config{}

	tests := []struct {
		key      string
		value    string
		expected interface{}
		field    func() interface{}
	}{
		{"prefix", "NEW-", "NEW-", func() interface{} { return cfg.Prefix }},
		{"suffix_prd", "-new-prod", "-new-prod", func() interface{} { return cfg.SuffixPrd }},
		{"suffix_hml", "-new-stage", "-new-stage", func() interface{} { return cfg.SuffixHml }},
		{"color", "false", false, func() interface{} { return cfg.Color }},
		{"color", "true", true, func() interface{} { return cfg.Color }},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			err := cfg.Set(tt.key, tt.value)
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}
			if got := tt.field(); got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestConfig_Set_InvalidKey(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("invalid_key", "value")
	if err == nil {
		t.Error("Expected error for invalid key, got nil")
	}
}

func TestConfig_Set_InvalidColorValue(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("color", "invalid")
	if err == nil {
		t.Error("Expected error for invalid color value, got nil")
	}
}