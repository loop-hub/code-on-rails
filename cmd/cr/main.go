package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loop-hub/code-on-rails/internal/analyzer"
	"github.com/loop-hub/code-on-rails/internal/config"
	"github.com/loop-hub/code-on-rails/internal/detector"
	"github.com/loop-hub/code-on-rails/internal/matcher"
	"github.com/loop-hub/code-on-rails/internal/reporter"
	"github.com/loop-hub/code-on-rails/pkg/patterns"
	"github.com/spf13/cobra"
)

var (
	verbose   bool
	aiModel   string
	threshold float64
	format    string
	repoURL   string
	commitSHA string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "cr",
		Short: "Code on Rails - Pattern enforcement for AI-generated code",
		Long: `Code on Rails learns your codebase patterns and ensures every AI-generated 
change fits your architecture. Works locally and in CI/CD.`,
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add commands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(learnCmd())
	rootCmd.AddCommand(blessCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	var language string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap patterns from existing codebase",
		Long:  `Analyze your codebase and automatically extract common patterns.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if config already exists
			if config.Exists("") {
				fmt.Println("Configuration file already exists. Use --force to overwrite.")
				return nil
			}

			// Auto-detect language if not specified
			if language == "" {
				language = detectLanguage(".")
			}

			fmt.Println("Initializing Code on Rails...")
			fmt.Printf("Language: %s\n\n", language)

			// Create analyzer
			a := analyzer.New(language)

			// Extract patterns
			patterns, err := a.ExtractPatterns(".")
			if err != nil {
				return fmt.Errorf("failed to extract patterns: %w", err)
			}

			// Create configuration
			cfg := config.NewDefault(language)
			cfg.Patterns = patterns

			// Save configuration
			if err := config.Save(cfg, ""); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			// Count total files discovered
			totalFiles := 0
			for _, p := range patterns {
				totalFiles += p.SeenCount
			}

			// Report results
			rep := reporter.New(verbose)
			rep.ReportInit(patterns, totalFiles, language)

			return nil
		},
	}

	cmd.Flags().StringVarP(&language, "language", "l", "", "programming language (auto-detected if not specified)")

	return cmd
}

func checkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [files...]",
		Short: "Check files against established patterns",
		Long:  `Validate AI-generated code against your codebase patterns.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.Load("")
			if err != nil {
				return fmt.Errorf("failed to load config: %w (run 'cr init' first)", err)
			}

			// Override threshold if specified
			if threshold > 0 {
				cfg.Settings.AutoApproveThreshold = threshold
			}

			// Detect language if not configured
			lang := cfg.Language
			if lang == "" {
				lang = detectLanguage(".")
			}

			// Get files to check
			files := args
			if len(files) == 0 {
				// Detect AI-generated files
				det := detector.NewWithLanguage(&cfg.Detection, lang)
				files, err = det.DetectFiles(".")
				if err != nil {
					return fmt.Errorf("failed to detect AI files: %w", err)
				}

				if len(files) == 0 {
					if format == "json" {
						fmt.Println(`{"summary":{"total_files":0},"auto_approved":[],"needs_review":[]}`)
					} else if format == "github" {
						fmt.Println("## 🤖 Code on Rails\n\n✨ No AI-generated code detected in this PR.")
					} else {
						fmt.Println("No AI-generated files found.")
						fmt.Printf("Detected language: %s\n", lang)
					}
					return nil
				}
			}

			// Create matcher
			m := matcher.New(cfg.Patterns, cfg.Settings.AutoApproveThreshold)

			// Match each file
			matches := []patterns.PatternMatch{}
			for _, file := range files {
				match, err := m.MatchFile(file)
				if err != nil {
					if verbose {
						fmt.Printf("Warning: failed to match %s: %v\n", file, err)
					}
					continue
				}
				matches = append(matches, *match)
			}

			// Report results based on format
			rep := reporter.New(verbose)

			// Get GitHub context from environment if not specified
			if repoURL == "" {
				repoURL = os.Getenv("GITHUB_REPOSITORY")
				if repoURL != "" {
					repoURL = "https://github.com/" + repoURL
				}
			}
			if commitSHA == "" {
				commitSHA = os.Getenv("GITHUB_SHA")
			}

			switch format {
			case "json":
				fmt.Println(rep.ReportJSON(matches, lang))
			case "github":
				fmt.Println(rep.FormatForGitHub(matches, repoURL, commitSHA))
			default:
				rep.Report(matches)
			}

			// Exit with error if any files failed (only in default mode)
			if format == "" {
				for _, match := range matches {
					if !match.AutoApprove {
						for _, dev := range match.Deviations {
							if dev.Severity == patterns.SeverityError {
								os.Exit(1)
							}
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&aiModel, "ai-model", "a", "", "filter by AI model (claude, copilot, cursor, any)")
	cmd.Flags().StringVarP(&format, "format", "f", "", "output format: json, github, or default (text)")
	cmd.Flags().StringVar(&repoURL, "repo-url", "", "GitHub repository URL (for github format links)")
	cmd.Flags().StringVar(&commitSHA, "sha", "", "Git commit SHA (for github format links)")
	cmd.Flags().Float64VarP(&threshold, "threshold", "t", 0, "auto-approve threshold (0-100)")

	return cmd
}

func learnCmd() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Update patterns from recently merged code",
		Long:  `Analyze recently merged code and update pattern definitions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.Load("")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Analyzing merged code from last %d days...\n", days)

			// Get recently merged files
			det := detector.New(&cfg.Detection)
			files, err := det.GetRecentAIFiles(".", days)
			if err != nil {
				return fmt.Errorf("failed to get recent files: %w", err)
			}

			if len(files) == 0 {
				fmt.Println("No recently merged AI-generated files found.")
				return nil
			}

			// Re-analyze to find new patterns
			a := analyzer.New(cfg.Language)
			newPatterns, err := a.ExtractPatterns(".")
			if err != nil {
				return fmt.Errorf("failed to extract patterns: %w", err)
			}

			// Merge new patterns with existing (updates cfg.Patterns in-place)
			updated := mergePatterns(cfg.Patterns, newPatterns)

			// Add any new patterns that weren't in existing config
			existingIDs := make(map[string]bool)
			for _, p := range cfg.Patterns {
				existingIDs[p.ID] = true
			}
			for _, newPat := range newPatterns {
				if !existingIDs[newPat.ID] {
					cfg.Patterns = append(cfg.Patterns, newPat)
					updated++
				}
			}

			// Save
			if err := config.Save(cfg, ""); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			// Report
			rep := reporter.New(verbose)
			rep.ReportLearn([]patterns.Pattern{}, updated)

			return nil
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 7, "number of days to look back")

	return cmd
}

func blessCmd() *cobra.Command {
	var reason string
	var weight float64

	cmd := &cobra.Command{
		Use:   "bless <file>",
		Short: "Mark a file as a blessed pattern example",
		Long: `Bless a file to elevate it as a high-quality pattern reference.
Blessed files have higher weight (1.5x by default) when matching patterns.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			// Verify file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", filePath)
			}

			// Load configuration
			cfg, err := config.Load("")
			if err != nil {
				return fmt.Errorf("failed to load config: %w (run 'cr init' first)", err)
			}

			// Find which pattern this file belongs to
			m := matcher.New(cfg.Patterns, cfg.Settings.AutoApproveThreshold)
			match, err := m.MatchFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to analyze file: %w", err)
			}

			if match.Pattern == nil {
				return fmt.Errorf("no matching pattern found for %s", filePath)
			}

			// Add to config_blessed for the matched pattern
			blessed := patterns.BlessedExample{
				Path:        filePath,
				BlessedBy:   "config",
				BlessedDate: time.Now(),
				Reason:      reason,
				Weight:      weight,
			}

			// Find and update the pattern
			for i, p := range cfg.Patterns {
				if p.ID == match.Pattern.ID {
					cfg.Patterns[i].ConfigBlessed = append(cfg.Patterns[i].ConfigBlessed, blessed)
					break
				}
			}

			// Save configuration
			if err := config.Save(cfg, ""); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("✓ Blessed %s\n", filePath)
			fmt.Printf("  Pattern: %s\n", match.Pattern.Name)
			fmt.Printf("  Weight: %.1fx\n", weight)
			if reason != "" {
				fmt.Printf("  Reason: %s\n", reason)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&reason, "reason", "r", "", "reason for blessing this file")
	cmd.Flags().Float64VarP(&weight, "weight", "w", 1.5, "weight multiplier for pattern matching")

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Code on Rails v0.1.0 (PoC)")
		},
	}
}

// Helper functions

func detectLanguage(path string) string {
	// Detect language based on project files
	if path == "" {
		path = "."
	}

	// Check for Go
	if fileExists(path + "/go.mod") {
		return "go"
	}

	// Check for TypeScript/React
	if fileExists(path + "/tsconfig.json") {
		// Check if it's a React project
		if fileExists(path+"/package.json") && fileContains(path+"/package.json", "react") {
			return "react"
		}
		return "typescript"
	}

	// Check for JavaScript/React
	if fileExists(path + "/package.json") {
		if fileContains(path+"/package.json", "react") {
			return "react"
		}
		return "javascript"
	}

	// Default to detecting based on file prevalence
	goCount := countFilesWithExtension(path, ".go")
	tsCount := countFilesWithExtension(path, ".ts") + countFilesWithExtension(path, ".tsx")
	jsCount := countFilesWithExtension(path, ".js") + countFilesWithExtension(path, ".jsx")

	if goCount >= tsCount && goCount >= jsCount && goCount > 0 {
		return "go"
	}
	if tsCount >= jsCount && tsCount > 0 {
		return "typescript"
	}
	if jsCount > 0 {
		return "javascript"
	}

	return "go" // Default fallback
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileContains(path string, substr string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), substr)
}

func countFilesWithExtension(dir string, ext string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ext) {
			count++
		}
		// Limit depth to avoid scanning too deep
		if d.IsDir() && strings.Count(path, string(os.PathSeparator)) > 5 {
			return filepath.SkipDir
		}
		return nil
	})
	return count
}

func mergePatterns(existing, new []patterns.Pattern) int {
	// Simple merge: count how many existing patterns got updated
	// In real implementation, would intelligently merge patterns
	updated := 0
	for i := range existing {
		for _, newPat := range new {
			if existing[i].ID == newPat.ID {
				existing[i].SeenCount = newPat.SeenCount
				existing[i].Confidence = newPat.Confidence
				updated++
				break
			}
		}
	}
	return updated
}
