package main

import (
	"fmt"
	"os"

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
			rep.ReportInit(patterns, totalFiles)

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

			// Get files to check
			files := args
			if len(files) == 0 {
				// Detect AI-generated files
				det := detector.New(&cfg.Detection)
				files, err = det.DetectFiles(".")
				if err != nil {
					return fmt.Errorf("failed to detect AI files: %w", err)
				}

				if len(files) == 0 {
					fmt.Println("No AI-generated files found.")
					fmt.Println("Hint: Tag commits with [ai], [claude], [copilot], etc.")
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

			// Report results
			rep := reporter.New(verbose)
			rep.Report(matches)

			// Exit with error if any files failed
			for _, match := range matches {
				if !match.AutoApprove {
					for _, dev := range match.Deviations {
						if dev.Severity == patterns.SeverityError {
							os.Exit(1)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&aiModel, "ai-model", "a", "", "filter by AI model (claude, copilot, cursor, any)")
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
	// Simple language detection based on files
	// In real implementation, would check file extensions
	return "go"
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
