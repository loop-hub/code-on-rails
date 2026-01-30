package matcher

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strings"

	"github.com/loop-hub/code-on-rails/pkg/patterns"
)

// Matcher matches code against patterns
type Matcher struct {
	Patterns  []patterns.Pattern
	Threshold float64
}

// New creates a new matcher
func New(pats []patterns.Pattern, threshold float64) *Matcher {
	return &Matcher{
		Patterns:  pats,
		Threshold: threshold,
	}
}

// MatchFile matches a file against all patterns using weighted hybrid approach
func (m *Matcher) MatchFile(filePath string) (*patterns.PatternMatch, error) {
	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Try to match against each pattern
	var bestMatch *patterns.PatternMatch
	bestScore := 0.0

	for _, pattern := range m.Patterns {
		if !m.shouldTryPattern(filePath, pattern) {
			continue
		}

		// Try annotated golden first (highest weight: 2.0x)
		if len(pattern.AnnotatedGolden) > 0 {
			for _, golden := range pattern.AnnotatedGolden {
				score, deviations := m.scoreAgainstGolden(file, golden, filePath)
				weightedScore := score * golden.Weight // 2.0x

				if weightedScore > bestScore {
					bestScore = weightedScore
					bestMatch = &patterns.PatternMatch{
						Pattern:       &pattern,
						FilePath:      filePath,
						Score:         score,
						WeightedScore: weightedScore,
						MatchType:     "annotated_golden",
						GoldenRef:     &golden,
						Deviations:    deviations,
						AutoApprove:   score >= m.Threshold,
					}
				}
			}
		}

		// Try config blessed (weight: 1.5x)
		if len(pattern.ConfigBlessed) > 0 {
			for _, blessed := range pattern.ConfigBlessed {
				score, deviations := m.scoreAgainstBlessed(file, blessed, filePath)
				weightedScore := score * blessed.Weight // 1.5x

				if weightedScore > bestScore {
					bestScore = weightedScore
					bestMatch = &patterns.PatternMatch{
						Pattern:       &pattern,
						FilePath:      filePath,
						Score:         score,
						WeightedScore: weightedScore,
						MatchType:     "config_blessed",
						BlessedRef:    &blessed,
						Deviations:    deviations,
						AutoApprove:   score >= m.Threshold,
					}
				}
			}
		}

		// Try discovered examples (weight: 1.0x)
		if len(pattern.Discovered) > 0 {
			for _, discovered := range pattern.Discovered {
				score, deviations := m.scoreAgainstDiscovered(file, discovered, filePath)
				weightedScore := score * discovered.Weight // 1.0x

				if weightedScore > bestScore {
					bestScore = weightedScore
					bestMatch = &patterns.PatternMatch{
						Pattern:       &pattern,
						FilePath:      filePath,
						Score:         score,
						WeightedScore: weightedScore,
						MatchType:     "discovered",
						DiscoveredRef: &discovered,
						Deviations:    deviations,
						AutoApprove:   score >= m.Threshold,
					}
				}
			}
		}
	}

	if bestMatch == nil {
		return &patterns.PatternMatch{
			FilePath:    filePath,
			Score:       0,
			MatchType:   "no_match",
			AutoApprove: false,
			Deviations: []patterns.Deviation{
				{
					Type:       patterns.DeviationNovel,
					Element:    "file",
					Severity:   patterns.SeverityWarning,
					Suggestion: "No matching pattern found. This may be a new pattern.",
				},
			},
		}, nil
	}

	return bestMatch, nil
}

// shouldTryPattern checks if a file might match a pattern
func (m *Matcher) shouldTryPattern(filePath string, pattern patterns.Pattern) bool {
	// Check file pattern
	if pattern.Detection.FilePattern != "" {
		// Simple glob matching
		patternGlob := strings.ReplaceAll(pattern.Detection.FilePattern, "*", "")
		if !strings.Contains(filePath, patternGlob) {
			return false
		}
	}

	// Check package path
	if pattern.Detection.PackagePath != "" {
		pkgPath := strings.ReplaceAll(pattern.Detection.PackagePath, "*", "")
		if !strings.Contains(filePath, pkgPath) {
			return false
		}
	}

	return true
}

// scoreAgainstGolden calculates similarity against a golden example
func (m *Matcher) scoreAgainstGolden(file *ast.File, golden patterns.GoldenExample, filePath string) (float64, []patterns.Deviation) {
	return m.scoreAgainstReference(file, golden.Path, filePath)
}

// scoreAgainstBlessed calculates similarity against a blessed example
func (m *Matcher) scoreAgainstBlessed(file *ast.File, blessed patterns.BlessedExample, filePath string) (float64, []patterns.Deviation) {
	return m.scoreAgainstReference(file, blessed.Path, filePath)
}

// scoreAgainstDiscovered calculates similarity against a discovered example
func (m *Matcher) scoreAgainstDiscovered(file *ast.File, discovered patterns.Example, filePath string) (float64, []patterns.Deviation) {
	return m.scoreAgainstReference(file, discovered.Path, filePath)
}

// scoreAgainstReference calculates similarity against a reference file
func (m *Matcher) scoreAgainstReference(file *ast.File, referencePath string, filePath string) (float64, []patterns.Deviation) {
	score := 100.0
	deviations := []patterns.Deviation{}

	// Check required imports from file
	fileImports := m.extractImports(file)

	// Parse reference file
	fset := token.NewFileSet()
	refFile, err := parser.ParseFile(fset, referencePath, nil, parser.ParseComments)
	if err != nil {
		return 50.0, deviations
	}

	refImports := m.extractImports(refFile)

	// Check import similarity
	for _, refImport := range refImports {
		if !contains(fileImports, refImport) {
			score -= 5.0
			deviations = append(deviations, patterns.Deviation{
				Type:       patterns.DeviationMissing,
				Element:    "import",
				Expected:   refImport,
				Severity:   patterns.SeverityWarning,
				Suggestion: fmt.Sprintf("Consider adding import: %s", refImport),
			})
		}
	}

	// Check for error handling patterns
	hasErrorHandling := m.checkErrorHandling(file)
	refHasErrorHandling := m.checkErrorHandling(refFile)

	if refHasErrorHandling && !hasErrorHandling {
		score -= 10.0
		deviations = append(deviations, patterns.Deviation{
			Type:       patterns.DeviationMissing,
			Element:    "error_handling",
			Expected:   "proper error handling",
			Severity:   patterns.SeverityWarning,
			Suggestion: "Add error handling like reference implementation",
		})
	}

	// Check structure similarity
	structureSimilarity := m.compareStructure(filePath, referencePath)
	score *= structureSimilarity

	return math.Max(0, score), deviations
}

// extractImports gets all imports from a file
func (m *Matcher) extractImports(file *ast.File) []string {
	imports := []string{}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, path)
	}
	return imports
}

// checkErrorHandling checks if file has proper error handling
func (m *Matcher) checkErrorHandling(file *ast.File) bool {
	hasErrorCheck := false
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for error checking patterns
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			if binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
				if binExpr.Op == token.NEQ {
					if ident, ok := binExpr.Y.(*ast.Ident); ok {
						if ident.Name == "nil" {
							hasErrorCheck = true
							return false
						}
					}
				}
			}
		}
		return true
	})
	return hasErrorCheck
}

// compareStructure compares structural similarity between files
func (m *Matcher) compareStructure(file1, file2 string) float64 {
	// Parse both files
	fset1 := token.NewFileSet()
	ast1, err1 := parser.ParseFile(fset1, file1, nil, 0)
	fset2 := token.NewFileSet()
	ast2, err2 := parser.ParseFile(fset2, file2, nil, 0)

	if err1 != nil || err2 != nil {
		return 0.5 // Default similarity if we can't parse
	}

	// Count node types in both files
	counts1 := m.countNodeTypes(ast1)
	counts2 := m.countNodeTypes(ast2)

	// Calculate similarity using cosine similarity
	return cosineSimilarity(counts1, counts2)
}

// countNodeTypes counts different AST node types
func (m *Matcher) countNodeTypes(file *ast.File) map[string]int {
	counts := make(map[string]int)
	ast.Inspect(file, func(n ast.Node) bool {
		if n != nil {
			nodeType := fmt.Sprintf("%T", n)
			counts[nodeType]++
		}
		return true
	})
	return counts
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func cosineSimilarity(a, b map[string]int) float64 {
	// Calculate dot product and magnitudes
	dotProduct := 0.0
	magA := 0.0
	magB := 0.0

	// Get all unique keys
	keys := make(map[string]bool)
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}

	for k := range keys {
		valA := float64(a[k])
		valB := float64(b[k])
		dotProduct += valA * valB
		magA += valA * valA
		magB += valB * valB
	}

	if magA == 0 || magB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(magA) * math.Sqrt(magB))
}
