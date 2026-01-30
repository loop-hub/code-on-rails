package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/loop-hub/code-on-rails/pkg/patterns"
)

// Annotation represents a code-on-rails annotation in source code
type Annotation struct {
	Type           string    // golden-example, anti-pattern, generated-from
	Pattern        string    // Pattern ID
	Version        string    // Pattern version
	Reason         string    // Why this is golden/anti-pattern
	Author         string    // GitHub handle
	BlessedDate    time.Time // When it was blessed
	QualityScore   int       // 0-100
	Supersedes     string    // Version this supersedes
	Deprecated     *time.Time
	MigrationGuide string
	GoldenExample  string // For generated-from
	GeneratedBy    string // AI tool that generated
	GeneratedDate  time.Time
	FunctionName   string // Function this annotation applies to
	LineNumber     int    // Line where annotation starts
}

// AnnotationParser parses code-on-rails annotations from source files
type AnnotationParser struct{}

// NewAnnotationParser creates a new annotation parser
func NewAnnotationParser() *AnnotationParser {
	return &AnnotationParser{}
}

// ParseFile parses all annotations in a file
func (p *AnnotationParser) ParseFile(filePath string) ([]Annotation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	annotations := []Annotation{}
	scanner := bufio.NewScanner(file)
	lineNum := 0
	inAnnotation := false
	currentAnnotation := Annotation{}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check if this is the start of an annotation
		if strings.HasPrefix(trimmed, "// @code-on-rails:") {
			inAnnotation = true
			currentAnnotation = Annotation{LineNumber: lineNum}

			// Parse the type
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				currentAnnotation.Type = strings.TrimSpace(parts[1])
			}
			continue
		}

		// If we're in an annotation block, parse fields
		if inAnnotation && strings.HasPrefix(trimmed, "// @") {
			p.parseAnnotationField(&currentAnnotation, trimmed)
			continue
		}

		// End of annotation block
		if inAnnotation && !strings.HasPrefix(trimmed, "//") {
			// Check if next line is a function declaration
			if strings.HasPrefix(trimmed, "func ") {
				currentAnnotation.FunctionName = p.extractFunctionName(trimmed)
			}
			annotations = append(annotations, currentAnnotation)
			inAnnotation = false
		}
	}

	return annotations, scanner.Err()
}

// parseAnnotationField parses a single annotation field
func (p *AnnotationParser) parseAnnotationField(ann *Annotation, line string) {
	// Remove leading "// @"
	line = strings.TrimPrefix(line, "// @")

	// Split on first colon
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "pattern":
		ann.Pattern = value
	case "version":
		ann.Version = value
	case "reason":
		ann.Reason = value
	case "author":
		ann.Author = value
	case "blessed":
		if t, err := time.Parse("2006-01-02", value); err == nil {
			ann.BlessedDate = t
		}
	case "quality-score":
		var score int
		fmt.Sscanf(value, "%d", &score)
		ann.QualityScore = score
	case "supersedes":
		ann.Supersedes = value
	case "migration-guide":
		ann.MigrationGuide = value
	case "golden-example":
		ann.GoldenExample = value
	case "generated-by":
		ann.GeneratedBy = value
	case "generated-date":
		if t, err := time.Parse("2006-01-02", value); err == nil {
			ann.GeneratedDate = t
		}
	case "deprecated":
		if t, err := time.Parse("2006-01-02", value); err == nil {
			ann.Deprecated = &t
		}
	}
}

// extractFunctionName extracts function name from function declaration
func (p *AnnotationParser) extractFunctionName(line string) string {
	// Pattern: func FunctionName( or func (receiver) FunctionName(
	re := regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// FindGoldenExamples finds all golden example annotations in a directory
func (p *AnnotationParser) FindGoldenExamples(rootPath string) ([]patterns.GoldenExample, error) {
	goldenExamples := []patterns.GoldenExample{}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse annotations in this file
		annotations, err := p.ParseFile(path)
		if err != nil {
			return nil // Skip files with parse errors
		}

		// Look for golden examples
		for _, ann := range annotations {
			if ann.Type == "golden-example" {
				goldenExamples = append(goldenExamples, patterns.GoldenExample{
					Path:         path,
					Function:     ann.FunctionName,
					Pattern:      ann.Pattern,
					Version:      ann.Version,
					BlessedBy:    ann.Author,
					BlessedDate:  ann.BlessedDate,
					Reason:       ann.Reason,
					QualityScore: ann.QualityScore,
					Weight:       2.0, // Golden examples get highest weight
				})
			}
		}

		return nil
	})

	return goldenExamples, err
}

// FindAntiPatterns finds all anti-pattern annotations
func (p *AnnotationParser) FindAntiPatterns(rootPath string) ([]patterns.AntiPattern, error) {
	antiPatterns := []patterns.AntiPattern{}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		annotations, err := p.ParseFile(path)
		if err != nil {
			return nil
		}

		for _, ann := range annotations {
			if ann.Type == "anti-pattern" {
				antiPatterns = append(antiPatterns, patterns.AntiPattern{
					Path:           path,
					Function:       ann.FunctionName,
					Pattern:        ann.Pattern,
					Reason:         ann.Reason,
					Deprecated:     ann.Deprecated,
					MigrationGuide: ann.MigrationGuide,
				})
			}
		}

		return nil
	})

	return antiPatterns, err
}
