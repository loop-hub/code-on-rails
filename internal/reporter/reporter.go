package reporter

import (
	"bufio"
	"encoding/json"
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

// JSONReport is the structured output for CI/CD systems
type JSONReport struct {
	Summary       ReportSummary     `json:"summary"`
	AutoApproved  []FileReport      `json:"auto_approved"`
	NeedsReview   []FileReport      `json:"needs_review"`
	NewPatterns   []string          `json:"new_patterns,omitempty"`
	Language      string            `json:"language"`
}

// ReportSummary contains aggregate statistics
type ReportSummary struct {
	TotalFiles      int `json:"total_files"`
	ApprovedFiles   int `json:"approved_files"`
	ReviewFiles     int `json:"review_files"`
	ApprovedLines   int `json:"approved_lines"`
	ReviewLines     int `json:"review_lines"`
	TimeSavedMins   int `json:"time_saved_mins"`
}

// FileReport represents a single file's analysis
type FileReport struct {
	FilePath     string            `json:"file_path"`
	Pattern      string            `json:"pattern"`
	PatternType  string            `json:"pattern_type"`
	Score        float64           `json:"score"`
	Lines        int               `json:"lines"`
	Deviations   []DeviationReport `json:"deviations,omitempty"`
	ReviewGuide  []string          `json:"review_guide,omitempty"`
}

// DeviationReport represents a single deviation
type DeviationReport struct {
	Element    string `json:"element"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion"`
	LineNumber int    `json:"line_number,omitempty"`
}

// ReportJSON outputs the analysis in JSON format
func (r *Reporter) ReportJSON(matches []patterns.PatternMatch, language string) string {
	report := JSONReport{
		Language:     language,
		AutoApproved: []FileReport{},
		NeedsReview:  []FileReport{},
	}

	for _, match := range matches {
		lines := estimateLines(match.FilePath)
		fileReport := FileReport{
			FilePath: match.FilePath,
			Score:    match.Score,
			Lines:    lines,
		}

		if match.Pattern != nil {
			fileReport.Pattern = match.Pattern.Name
			fileReport.PatternType = string(match.Pattern.Type)
		}

		if match.AutoApprove {
			report.AutoApproved = append(report.AutoApproved, fileReport)
			report.Summary.ApprovedFiles++
			report.Summary.ApprovedLines += lines
		} else {
			// Add deviations
			for _, dev := range match.Deviations {
				fileReport.Deviations = append(fileReport.Deviations, DeviationReport{
					Element:    dev.Element,
					Expected:   dev.Expected,
					Actual:     dev.Actual,
					Severity:   string(dev.Severity),
					Suggestion: dev.Suggestion,
					LineNumber: dev.LineNumber,
				})
			}

			// Add pattern-specific review guidance
			if match.Pattern != nil {
				fileReport.ReviewGuide = getPatternReviewGuide(match.Pattern.Type)
			}

			report.NeedsReview = append(report.NeedsReview, fileReport)
			report.Summary.ReviewFiles++
			report.Summary.ReviewLines += lines
		}
	}

	report.Summary.TotalFiles = len(matches)
	report.Summary.TimeSavedMins = report.Summary.ApprovedLines / 20

	jsonBytes, _ := json.MarshalIndent(report, "", "  ")
	return string(jsonBytes)
}

// getPatternReviewGuide returns review checklist based on pattern type
func getPatternReviewGuide(patternType patterns.PatternType) []string {
	switch patternType {
	case patterns.PatternComponent:
		return []string{
			"Verify props interface is complete and typed",
			"Check for proper error boundaries",
			"Ensure useEffect cleanup functions exist",
			"Validate accessibility (aria labels, keyboard nav)",
		}
	case patterns.PatternHook:
		return []string{
			"Verify hook follows rules of hooks",
			"Check dependency arrays are complete",
			"Ensure cleanup on unmount",
			"Validate return type consistency",
		}
	case patterns.PatternAPI:
		return []string{
			"Verify input validation and sanitization",
			"Check error handling and status codes",
			"Ensure authentication/authorization checks",
			"Validate response types match API contract",
		}
	case patterns.PatternService:
		return []string{
			"Verify error handling is comprehensive",
			"Check for proper logging",
			"Ensure dependencies are injected",
			"Validate business logic edge cases",
		}
	case patterns.PatternHTTPHandler:
		return []string{
			"Verify request validation",
			"Check error responses are consistent",
			"Ensure proper HTTP status codes",
			"Validate authentication middleware",
		}
	case patterns.PatternStore:
		return []string{
			"Verify state immutability",
			"Check for proper action typing",
			"Ensure selectors are memoized",
			"Validate async action handling",
		}
	default:
		return []string{
			"Verify code follows team conventions",
			"Check error handling",
			"Ensure proper documentation",
		}
	}
}

// FormatForGitHub formats results for GitHub PR comment with links
func (r *Reporter) FormatForGitHub(matches []patterns.PatternMatch, repoURL, sha string) string {
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

	sb.WriteString("## 🤖 Code on Rails - AI Code Review\n\n")

	// Summary stats at top
	totalLines := 0
	approvedLines := 0
	for _, m := range matches {
		lines := estimateLines(m.FilePath)
		totalLines += lines
		if m.AutoApprove {
			approvedLines += lines
		}
	}

	sb.WriteString(fmt.Sprintf("**%d files** analyzed | **%d** auto-approved | **%d** need review\n\n",
		len(matches), len(approvedFiles), len(reviewFiles)))

	// Approved section (collapsible)
	if len(approvedFiles) > 0 {
		sb.WriteString("<details>\n<summary>✅ <strong>Auto-approved</strong> (")
		sb.WriteString(fmt.Sprintf("%d files, %d lines)", len(approvedFiles), approvedLines))
		sb.WriteString("</summary>\n\n")
		sb.WriteString("These files follow established patterns and need minimal review:\n\n")
		sb.WriteString("| File | Pattern | Match |\n")
		sb.WriteString("|------|---------|-------|\n")
		for _, match := range approvedFiles {
			fileLink := formatGitHubLink(repoURL, sha, match.FilePath, 0)
			patternName := "unknown"
			if match.Pattern != nil {
				patternName = match.Pattern.Name
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %.0f%% |\n", fileLink, patternName, match.Score))
		}
		sb.WriteString("\n</details>\n\n")
	}

	// Review section (expanded)
	if len(reviewFiles) > 0 {
		sb.WriteString("### 🔍 Needs Human Review\n\n")
		sb.WriteString("These files have deviations from patterns or introduce new code that warrants review:\n\n")

		for _, match := range reviewFiles {
			patternName := "unknown"
			patternType := patterns.PatternUtil
			if match.Pattern != nil {
				patternName = match.Pattern.Name
				patternType = match.Pattern.Type
			}

			fileLink := formatGitHubLink(repoURL, sha, match.FilePath, 0)
			sb.WriteString(fmt.Sprintf("#### %s\n\n", fileLink))
			sb.WriteString(fmt.Sprintf("**Pattern:** `%s` (%.0f%% match)\n\n", patternName, match.Score))

			// Deviations with links to specific lines
			if len(match.Deviations) > 0 {
				sb.WriteString("**Issues found:**\n\n")
				for _, dev := range match.Deviations {
					icon := "⚠️"
					if dev.Severity == patterns.SeverityError {
						icon = "❌"
					} else if dev.Severity == patterns.SeverityInfo {
						icon = "ℹ️"
					}

					if dev.LineNumber > 0 {
						lineLink := formatGitHubLink(repoURL, sha, match.FilePath, dev.LineNumber)
						sb.WriteString(fmt.Sprintf("- %s **%s** at %s\n", icon, dev.Element, lineLink))
					} else {
						sb.WriteString(fmt.Sprintf("- %s **%s**\n", icon, dev.Element))
					}

					if dev.Expected != "" {
						sb.WriteString(fmt.Sprintf("  - Expected: `%s`\n", dev.Expected))
					}
					if dev.Suggestion != "" {
						sb.WriteString(fmt.Sprintf("  - 💡 %s\n", dev.Suggestion))
					}
				}
				sb.WriteString("\n")
			}

			// Pattern-specific review checklist
			reviewGuide := getPatternReviewGuide(patternType)
			if len(reviewGuide) > 0 {
				sb.WriteString("<details>\n<summary>📋 Review checklist for <code>")
				sb.WriteString(patternName)
				sb.WriteString("</code></summary>\n\n")
				for _, item := range reviewGuide {
					sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
				}
				sb.WriteString("\n</details>\n\n")
			}
		}
	}

	// No files case
	if len(matches) == 0 {
		sb.WriteString("✨ No AI-generated code detected in this PR.\n\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("_Generated by [Code on Rails](https://github.com/loop-hub/code-on-rails)_ • ")
	sb.WriteString(fmt.Sprintf("Review time saved: ~%d min\n", approvedLines/20))

	return sb.String()
}

// formatGitHubLink creates a GitHub link to a file or specific line
func formatGitHubLink(repoURL, sha, filePath string, lineNumber int) string {
	if repoURL == "" || sha == "" {
		return fmt.Sprintf("`%s`", filePath)
	}

	// Clean up repo URL
	repoURL = strings.TrimSuffix(repoURL, ".git")

	if lineNumber > 0 {
		return fmt.Sprintf("[`%s#L%d`](%s/blob/%s/%s#L%d)", filePath, lineNumber, repoURL, sha, filePath, lineNumber)
	}
	return fmt.Sprintf("[`%s`](%s/blob/%s/%s)", filePath, repoURL, sha, filePath)
}

// AIFeedback represents structured feedback for AI assistants like Claude
type AIFeedback struct {
	Summary         AIFeedbackSummary `json:"summary"`
	FilesToFix      []AIFileFeedback  `json:"files_to_fix"`
	ApprovedFiles   []string          `json:"approved_files"`
	PatternExamples []AIPatternRef    `json:"pattern_examples"`
	Instructions    string            `json:"instructions"`
}

// AIFeedbackSummary provides high-level context
type AIFeedbackSummary struct {
	TotalFiles      int    `json:"total_files"`
	NeedsFixes      int    `json:"needs_fixes"`
	AutoApproved    int    `json:"auto_approved"`
	PrimaryLanguage string `json:"primary_language"`
}

// AIFileFeedback describes issues in a specific file
type AIFileFeedback struct {
	FilePath       string             `json:"file_path"`
	PatternType    string             `json:"pattern_type"`
	MatchScore     float64            `json:"match_score"`
	Issues         []AIIssue          `json:"issues"`
	ReferenceFile  string             `json:"reference_file,omitempty"`
	ExpectedImports []string          `json:"expected_imports,omitempty"`
}

// AIIssue describes a single issue that needs fixing
type AIIssue struct {
	Type       string `json:"type"`        // "missing", "different", "novel"
	Element    string `json:"element"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	Suggestion string `json:"suggestion"`
	LineNumber int    `json:"line_number,omitempty"`
	Severity   string `json:"severity"` // "error", "warning", "info"
}

// AIPatternRef provides example code for a pattern
type AIPatternRef struct {
	PatternType string   `json:"pattern_type"`
	PatternName string   `json:"pattern_name"`
	ExampleFile string   `json:"example_file"`
	KeyElements []string `json:"key_elements"`
}

// FormatAIFeedback generates AI-readable feedback that Claude can act upon
func (r *Reporter) FormatAIFeedback(matches []patterns.PatternMatch, language string, patternList []patterns.Pattern) string {
	feedback := AIFeedback{
		Summary: AIFeedbackSummary{
			TotalFiles:      len(matches),
			PrimaryLanguage: language,
		},
		FilesToFix:      []AIFileFeedback{},
		ApprovedFiles:   []string{},
		PatternExamples: []AIPatternRef{},
	}

	// Collect pattern examples for reference
	patternExamples := make(map[string]AIPatternRef)
	for _, p := range patternList {
		// Find best example file for this pattern
		var exampleFile string
		if len(p.AnnotatedGolden) > 0 {
			exampleFile = p.AnnotatedGolden[0].Path
		} else if len(p.ConfigBlessed) > 0 {
			exampleFile = p.ConfigBlessed[0].Path
		} else if len(p.Discovered) > 0 {
			exampleFile = p.Discovered[0].Path
		}

		if exampleFile != "" {
			keyElements := []string{}
			for _, elem := range p.Structure.Required {
				keyElements = append(keyElements, elem)
			}
			patternExamples[string(p.Type)] = AIPatternRef{
				PatternType: string(p.Type),
				PatternName: p.Name,
				ExampleFile: exampleFile,
				KeyElements: keyElements,
			}
		}
	}

	for _, match := range matches {
		if match.AutoApprove {
			feedback.ApprovedFiles = append(feedback.ApprovedFiles, match.FilePath)
			feedback.Summary.AutoApproved++
		} else {
			fileFeedback := AIFileFeedback{
				FilePath:   match.FilePath,
				MatchScore: match.Score,
				Issues:     []AIIssue{},
			}

			if match.Pattern != nil {
				fileFeedback.PatternType = string(match.Pattern.Type)

				// Get reference file
				if match.GoldenRef != nil {
					fileFeedback.ReferenceFile = match.GoldenRef.Path
				} else if match.BlessedRef != nil {
					fileFeedback.ReferenceFile = match.BlessedRef.Path
				} else if match.DiscoveredRef != nil {
					fileFeedback.ReferenceFile = match.DiscoveredRef.Path
				}

				// Add expected imports from structure
				for _, elem := range match.Pattern.Structure.Elements {
					if elem.Type == patterns.ElementImport {
						fileFeedback.ExpectedImports = append(fileFeedback.ExpectedImports, elem.Name)
					}
				}

				// Add pattern example if not already present
				if _, exists := patternExamples[string(match.Pattern.Type)]; !exists && fileFeedback.ReferenceFile != "" {
					patternExamples[string(match.Pattern.Type)] = AIPatternRef{
						PatternType: string(match.Pattern.Type),
						PatternName: match.Pattern.Name,
						ExampleFile: fileFeedback.ReferenceFile,
					}
				}
			}

			// Convert deviations to issues
			for _, dev := range match.Deviations {
				issue := AIIssue{
					Type:       string(dev.Type),
					Element:    dev.Element,
					Expected:   dev.Expected,
					Actual:     dev.Actual,
					Suggestion: dev.Suggestion,
					LineNumber: dev.LineNumber,
					Severity:   string(dev.Severity),
				}
				fileFeedback.Issues = append(fileFeedback.Issues, issue)
			}

			feedback.FilesToFix = append(feedback.FilesToFix, fileFeedback)
			feedback.Summary.NeedsFixes++
		}
	}

	// Convert pattern examples map to slice
	for _, example := range patternExamples {
		feedback.PatternExamples = append(feedback.PatternExamples, example)
	}

	// Generate instructions for AI
	feedback.Instructions = generateAIInstructions(feedback)

	jsonBytes, _ := json.MarshalIndent(feedback, "", "  ")
	return string(jsonBytes)
}

// generateAIInstructions creates clear instructions for Claude/AI to follow
func generateAIInstructions(feedback AIFeedback) string {
	if feedback.Summary.NeedsFixes == 0 {
		return "All files follow established patterns. No changes required."
	}

	var sb strings.Builder
	sb.WriteString("INSTRUCTIONS FOR FIXING CODE:\n\n")
	sb.WriteString("1. Read each reference_file to understand the expected pattern\n")
	sb.WriteString("2. For each file in files_to_fix:\n")
	sb.WriteString("   - Compare against the reference pattern\n")
	sb.WriteString("   - Apply the suggestions from each issue\n")
	sb.WriteString("   - Ensure all expected_imports are present\n")
	sb.WriteString("3. After making changes, run 'cr check' to verify fixes\n\n")

	sb.WriteString("PRIORITY:\n")
	errorCount := 0
	warningCount := 0
	for _, file := range feedback.FilesToFix {
		for _, issue := range file.Issues {
			if issue.Severity == "error" {
				errorCount++
			} else if issue.Severity == "warning" {
				warningCount++
			}
		}
	}

	if errorCount > 0 {
		sb.WriteString(fmt.Sprintf("- Fix %d error(s) first (blocking issues)\n", errorCount))
	}
	if warningCount > 0 {
		sb.WriteString(fmt.Sprintf("- Then address %d warning(s)\n", warningCount))
	}

	return sb.String()
}

// FormatSkillFile generates a skill file that can be used across projects
func (r *Reporter) FormatSkillFile(patternList []patterns.Pattern, language string) string {
	skillFile := SkillFile{
		Version:  "1.0",
		Language: language,
		Skills:   []Skill{},
	}

	for _, p := range patternList {
		skill := Skill{
			ID:          p.ID,
			Name:        p.Name,
			Type:        string(p.Type),
			Description: generateSkillDescription(p),
			Detection: SkillDetection{
				FilePattern:   p.Detection.FilePattern,
				FuncPattern:   p.Detection.FuncPattern,
				StructPattern: p.Detection.StructPattern,
			},
			Structure: SkillStructure{
				Required: p.Structure.Required,
				Optional: p.Structure.Optional,
			},
			Examples: []string{},
		}

		// Add example files
		for _, g := range p.AnnotatedGolden {
			skill.Examples = append(skill.Examples, g.Path)
		}
		for _, b := range p.ConfigBlessed {
			skill.Examples = append(skill.Examples, b.Path)
		}
		// Limit discovered examples to prevent bloat
		maxDiscovered := 3
		for i, d := range p.Discovered {
			if i >= maxDiscovered {
				break
			}
			skill.Examples = append(skill.Examples, d.Path)
		}

		skillFile.Skills = append(skillFile.Skills, skill)
	}

	jsonBytes, _ := json.MarshalIndent(skillFile, "", "  ")
	return string(jsonBytes)
}

// SkillFile represents a portable skill definition
type SkillFile struct {
	Version  string  `json:"version"`
	Language string  `json:"language"`
	Skills   []Skill `json:"skills"`
}

// Skill represents a single pattern skill
type Skill struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Detection   SkillDetection `json:"detection"`
	Structure   SkillStructure `json:"structure"`
	Examples    []string       `json:"examples"`
}

// SkillDetection defines how to identify this pattern
type SkillDetection struct {
	FilePattern   string `json:"file_pattern,omitempty"`
	FuncPattern   string `json:"func_pattern,omitempty"`
	StructPattern string `json:"struct_pattern,omitempty"`
}

// SkillStructure defines what the pattern should contain
type SkillStructure struct {
	Required []string `json:"required,omitempty"`
	Optional []string `json:"optional,omitempty"`
}

// generateSkillDescription creates a human-readable description of the pattern
func generateSkillDescription(p patterns.Pattern) string {
	var desc string
	switch p.Type {
	case patterns.PatternHTTPHandler:
		desc = "HTTP handler pattern for processing web requests"
	case patterns.PatternService:
		desc = "Service layer pattern for business logic"
	case patterns.PatternRepository:
		desc = "Repository pattern for data access"
	case patterns.PatternMiddleware:
		desc = "Middleware pattern for request/response processing"
	case patterns.PatternComponent:
		desc = "React component pattern for UI elements"
	case patterns.PatternHook:
		desc = "Custom React hook pattern for reusable stateful logic"
	case patterns.PatternAPI:
		desc = "API route handler pattern for backend endpoints"
	case patterns.PatternStore:
		desc = "State management pattern for global state"
	default:
		desc = fmt.Sprintf("Pattern for %s code organization", p.Name)
	}
	return desc
}
