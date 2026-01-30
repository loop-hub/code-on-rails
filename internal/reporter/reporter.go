package reporter

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/loop-hub/code-on-rails/pkg/patterns"
)

// Reporter formats analysis results
type Reporter struct {
	Verbose bool
}

// New creates a new reporter
func New(verbose bool) *Reporter {
	return &Reporter{Verbose: verbose}
}

// Report prints pattern match results
func (r *Reporter) Report(matches []patterns.PatternMatch) {
	if len(matches) == 0 {
		fmt.Println("No files to check.")
		return
	}

	fmt.Println("\nAnalyzing AI-generated code...\n")

	approvedCount := 0
	approvedLines := 0
	warningCount := 0
	warningLines := 0
	errorCount := 0
	errorLines := 0

	for _, match := range matches {
		r.printMatch(match)

		// Count stats
		lines := estimateLines(match.FilePath)
		if match.AutoApprove {
			approvedCount++
			approvedLines += lines
		} else {
			hasError := false
			for _, dev := range match.Deviations {
				if dev.Severity == patterns.SeverityError {
					hasError = true
					break
				}
			}
			if hasError {
				errorCount++
				errorLines += lines
			} else {
				warningCount++
				warningLines += lines
			}
		}
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Summary:")
	fmt.Printf("  ✓ %d file(s) auto-approved (%d lines)\n", approvedCount, approvedLines)
	if warningCount > 0 {
		fmt.Printf("  ⚠ %d file(s) need review (%d lines)\n", warningCount, warningLines)
	}
	if errorCount > 0 {
		fmt.Printf("  ✗ %d file(s) have errors (%d lines)\n", errorCount, errorLines)
	}

	estimatedTime := (warningLines + errorLines) / 20 // Assume 20 lines/minute review
	if estimatedTime > 0 {
		fmt.Printf("\nEstimated review time saved: %d minutes\n", approvedLines/20)
	}
}

// printMatch prints a single match result
func (r *Reporter) printMatch(match patterns.PatternMatch) {
	if match.AutoApprove {
		fmt.Printf("✓ %s\n", match.FilePath)
		if match.Pattern != nil {
			fmt.Printf("  Pattern: %s (%.0f%% match)\n", match.Pattern.Name, match.Score)

			// Show reference based on match type
			switch match.MatchType {
			case "annotated_golden":
				if match.GoldenRef != nil {
					fmt.Printf("  Golden example: %s\n", match.GoldenRef.Path)
					if match.GoldenRef.BlessedBy != "" {
						fmt.Printf("  Blessed by: %s\n", match.GoldenRef.BlessedBy)
					}
					if match.GoldenRef.Reason != "" {
						fmt.Printf("  Reason: %s\n", match.GoldenRef.Reason)
					}
				}
			case "config_blessed":
				if match.BlessedRef != nil {
					fmt.Printf("  Reference: %s (blessed)\n", match.BlessedRef.Path)
				}
			case "discovered":
				if match.DiscoveredRef != nil {
					fmt.Printf("  Reference: %s\n", match.DiscoveredRef.Path)
				}
			}
		}
		fmt.Println("  Auto-approved\n")
	} else {
		// Determine icon based on severity
		icon := "⚠"
		for _, dev := range match.Deviations {
			if dev.Severity == patterns.SeverityError {
				icon = "✗"
				break
			}
		}

		fmt.Printf("%s %s\n", icon, match.FilePath)
		if match.Pattern != nil {
			fmt.Printf("  Pattern: %s (%.0f%% match)\n", match.Pattern.Name, match.Score)

			// Show reference based on match type
			switch match.MatchType {
			case "annotated_golden":
				if match.GoldenRef != nil {
					fmt.Printf("  Golden example: %s\n", match.GoldenRef.Path)
					if match.GoldenRef.BlessedBy != "" {
						fmt.Printf("  Blessed by: %s\n", match.GoldenRef.BlessedBy)
					}
				}
			case "config_blessed":
				if match.BlessedRef != nil {
					fmt.Printf("  Reference: %s (blessed)\n", match.BlessedRef.Path)
				}
			case "discovered":
				if match.DiscoveredRef != nil {
					fmt.Printf("  Reference: %s\n", match.DiscoveredRef.Path)
				}
			}
		}

		if len(match.Deviations) > 0 {
			fmt.Println("  Deviations:")
			for _, dev := range match.Deviations {
				r.printDeviation(dev)
			}
		}
		fmt.Println()
	}
}

// printDeviation prints a single deviation
func (r *Reporter) printDeviation(dev patterns.Deviation) {
	icon := "•"
	switch dev.Severity {
	case patterns.SeverityError:
		icon = "✗"
	case patterns.SeverityWarning:
		icon = "⚠"
	case patterns.SeverityInfo:
		icon = "ℹ"
	}

	fmt.Printf("    %s %s", icon, dev.Element)
	if dev.Expected != "" {
		fmt.Printf(" (expected: %s", dev.Expected)
		if dev.Actual != "" {
			fmt.Printf(", found: %s", dev.Actual)
		}
		fmt.Print(")")
	}
	fmt.Println()

	if dev.Suggestion != "" {
		fmt.Printf("      Suggestion: %s\n", dev.Suggestion)
	}
}

// ReportInit prints initialization results
func (r *Reporter) ReportInit(patterns []patterns.Pattern, totalFiles int, language string) {
	fmt.Println("Scanning codebase...")
	lang := "code"
	switch language {
	case "go":
		lang = "Go"
	case "typescript", "ts":
		lang = "TypeScript"
	case "javascript", "js":
		lang = "JavaScript"
	case "react":
		lang = "React/TypeScript"
	}
	fmt.Printf("→ Discovered %d %s files\n", totalFiles, lang)
	fmt.Println("→ Identified patterns:")

	for _, p := range patterns {
		fmt.Printf("  • %d %s", p.SeenCount, p.Name)
		if p.SeenCount != 1 {
			fmt.Print("s")
		}
		fmt.Println()
	}

	fmt.Println("→ Generated .code-on-rails.yml")
	fmt.Println("✓ Ready to use!")
}

// ReportLearn prints learning results
func (r *Reporter) ReportLearn(newPatterns []patterns.Pattern, updatedPatterns int) {
	fmt.Println("Analyzing merged code from last week...")

	if len(newPatterns) > 0 {
		fmt.Printf("→ Discovered %d new pattern(s):\n", len(newPatterns))
		for _, p := range newPatterns {
			fmt.Printf("  • %s (%d examples)\n", p.Name, p.SeenCount)
		}
	}

	if updatedPatterns > 0 {
		fmt.Printf("→ Updated %d existing pattern(s)\n", updatedPatterns)
	}

	if len(newPatterns) == 0 && updatedPatterns == 0 {
		fmt.Println("→ No new patterns found")
	}

	fmt.Println("✓ Configuration updated")
}

// estimateLines counts the number of lines in a file
func estimateLines(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 100 // fallback to estimate
	}
	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}
	if lineCount == 0 {
		return 100 // fallback for empty/error
	}
	return lineCount
}

// FormatForGitHub formats results for GitHub PR comment
func (r *Reporter) FormatForGitHub(matches []patterns.PatternMatch) string {
	var sb strings.Builder

	approvedFiles := []patterns.PatternMatch{}
	reviewFiles := []patterns.PatternMatch{}

	for _, match := range matches {
		if match.AutoApprove {
			approvedFiles = append(approvedFiles, match)
		} else {
			reviewFiles = append(reviewFiles, match)
		}
	}

	// Approved section
	if len(approvedFiles) > 0 {
		sb.WriteString("## ✅ Auto-approved (" + fmt.Sprintf("%d files", len(approvedFiles)) + ")\n\n")
		for _, match := range approvedFiles {
			sb.WriteString(fmt.Sprintf("- `%s` (%.0f%% match)\n", match.FilePath, match.Score))
		}
		sb.WriteString("\n")
	}

	// Review section
	if len(reviewFiles) > 0 {
		sb.WriteString("## ⚠️ Needs Review (" + fmt.Sprintf("%d files", len(reviewFiles)) + ")\n\n")
		for _, match := range reviewFiles {
			sb.WriteString(fmt.Sprintf("### `%s`\n\n", match.FilePath))
			if match.Pattern != nil {
				sb.WriteString(fmt.Sprintf("**Pattern:** %s (%.0f%% match)\n\n", match.Pattern.Name, match.Score))
			}
			if len(match.Deviations) > 0 {
				sb.WriteString("**Deviations:**\n")
				for _, dev := range match.Deviations {
					sb.WriteString(fmt.Sprintf("- %s: %s\n", dev.Element, dev.Suggestion))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Stats
	sb.WriteString("## 📊 Stats\n\n")
	sb.WriteString(fmt.Sprintf("- %d lines auto-approved\n", len(approvedFiles)*100))
	sb.WriteString(fmt.Sprintf("- %d lines flagged for review\n", len(reviewFiles)*100))

	return sb.String()
}
