package config

import (
	"fmt"
	"os"

	"github.com/loop-hub/code-on-rails/pkg/patterns"
	"gopkg.in/yaml.v3"
)

const ConfigFileName = ".code-on-rails.yml"

// Config represents the full configuration
type Config struct {
	Version   string             `yaml:"version"`
	Language  string             `yaml:"language"`
	AISource  string             `yaml:"ai_source"`
	Patterns  []patterns.Pattern `yaml:"patterns"`
	Settings  Settings           `yaml:"settings"`
	Detection DetectionConfig    `yaml:"detection"`
}

// Settings for pattern matching behavior
type Settings struct {
	AutoApproveThreshold float64 `yaml:"auto_approve_threshold"`
	LearnOnMerge         bool    `yaml:"learn_on_merge"`
}

// DetectionConfig for AI code detection
type DetectionConfig struct {
	Method         string   `yaml:"method"` // commit_message, git_notes, heuristic, all
	CommitPrefixes []string `yaml:"commit_prefixes"`
}

// Load reads configuration from file
func Load(path string) (*Config, error) {
	if path == "" {
		path = ConfigFileName
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if cfg.Settings.AutoApproveThreshold == 0 {
		cfg.Settings.AutoApproveThreshold = 95.0
	}
	if cfg.AISource == "" {
		cfg.AISource = "any"
	}
	if cfg.Detection.Method == "" {
		cfg.Detection.Method = "commit_message"
	}
	if len(cfg.Detection.CommitPrefixes) == 0 {
		cfg.Detection.CommitPrefixes = []string{"[ai", "[claude", "[copilot", "[cursor"}
	}

	return &cfg, nil
}

// Save writes configuration to file
func Save(cfg *Config, path string) error {
	if path == "" {
		path = ConfigFileName
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Exists checks if config file exists
func Exists(path string) bool {
	if path == "" {
		path = ConfigFileName
	}
	_, err := os.Stat(path)
	return err == nil
}

// NewDefault creates a default configuration
func NewDefault(language string) *Config {
	return &Config{
		Version:  "1.0",
		Language: language,
		AISource: "any",
		Patterns: []patterns.Pattern{},
		Settings: Settings{
			AutoApproveThreshold: 95.0,
			LearnOnMerge:         true,
		},
		Detection: DetectionConfig{
			Method:         "commit_message",
			CommitPrefixes: []string{"[ai", "[claude", "[copilot", "[cursor"},
		},
	}
}
