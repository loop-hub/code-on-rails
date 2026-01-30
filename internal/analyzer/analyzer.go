package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loop-hub/code-on-rails/pkg/patterns"
)

// Analyzer extracts patterns from codebases
type Analyzer struct {
	Language string
}

// New creates a new analyzer
func New(language string) *Analyzer {
	return &Analyzer{Language: language}
}

// ExtractPatterns analyzes a codebase and extracts common patterns
func (a *Analyzer) ExtractPatterns(rootPath string) ([]patterns.Pattern, error) {
	if a.Language != "go" {
		return nil, fmt.Errorf("unsupported language: %s", a.Language)
	}

	// Step 1: Find annotated golden examples
	parser := NewAnnotationParser()
	goldenExamples, err := parser.FindGoldenExamples(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find golden examples: %w", err)
	}

	// Find anti-patterns
	antiPatterns, err := parser.FindAntiPatterns(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find anti-patterns: %w", err)
	}

	// Step 2: Find all Go files for discovery
	files, err := findGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	// Parse all files
	fileInfos := make([]patterns.FileInfo, 0, len(files))
	for _, file := range files {
		info, err := parseGoFile(file)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}
		fileInfos = append(fileInfos, *info)
	}

	// Group files by structural similarity
	groups := groupByStructure(fileInfos)

	// Extract patterns from groups
	extractedPatterns := []patterns.Pattern{}

	// Organize golden examples by pattern
	goldenByPattern := make(map[string][]patterns.GoldenExample)
	for _, golden := range goldenExamples {
		goldenByPattern[golden.Pattern] = append(goldenByPattern[golden.Pattern], golden)
	}

	// Organize anti-patterns by pattern
	antiByPattern := make(map[string][]patterns.AntiPattern)
	for _, anti := range antiPatterns {
		antiByPattern[anti.Pattern] = append(antiByPattern[anti.Pattern], anti)
	}

	for patternType, group := range groups {
		if len(group) < 3 && len(goldenByPattern[string(patternType)]) == 0 {
			// Need at least 3 examples to call it a pattern, unless we have golden examples
			continue
		}

		pattern := extractPattern(patternType, group)

		// Add golden examples if they exist for this pattern
		if goldens, ok := goldenByPattern[string(patternType)]; ok {
			pattern.AnnotatedGolden = goldens
		}

		// Add anti-patterns if they exist
		if antis, ok := antiByPattern[string(patternType)]; ok {
			pattern.AntiPatterns = antis
		}

		// Convert regular examples to discovered format
		pattern.Discovered = make([]patterns.Example, 0, len(group))
		for _, file := range group {
			// Skip if this file is already a golden example
			isGolden := false
			for _, golden := range pattern.AnnotatedGolden {
				if golden.Path == file.Path {
					isGolden = true
					break
				}
			}
			if !isGolden {
				pattern.Discovered = append(pattern.Discovered, patterns.Example{
					Path:            file.Path,
					SimilarityScore: 0.9, // Placeholder
					Weight:          1.0,
				})
			}
		}

		extractedPatterns = append(extractedPatterns, pattern)
	}

	return extractedPatterns, nil
}

// findGoFiles recursively finds all .go files
func findGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			// Skip test files and vendor
			if !strings.HasSuffix(path, "_test.go") && !strings.Contains(path, "/vendor/") {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

// parseGoFile parses a Go file and extracts structure
func parseGoFile(filePath string) (*patterns.FileInfo, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	info := &patterns.FileInfo{
		Path:      filePath,
		Package:   file.Name.Name,
		Imports:   []string{},
		Functions: []patterns.FunctionInfo{},
		Types:     []patterns.TypeInfo{},
	}

	// Extract imports
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		info.Imports = append(info.Imports, path)
	}

	// Extract functions and types
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			funcInfo := patterns.FunctionInfo{
				Name: node.Name.Name,
			}
			if node.Recv != nil && len(node.Recv.List) > 0 {
				// It's a method
				if starExpr, ok := node.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						funcInfo.Receiver = ident.Name
					}
				}
			}
			info.Functions = append(info.Functions, funcInfo)

		case *ast.TypeSpec:
			typeInfo := patterns.TypeInfo{
				Name: node.Name.Name,
			}
			if structType, ok := node.Type.(*ast.StructType); ok {
				typeInfo.Kind = "struct"
				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						typeInfo.Fields = append(typeInfo.Fields, name.Name)
					}
				}
			} else if _, ok := node.Type.(*ast.InterfaceType); ok {
				typeInfo.Kind = "interface"
			}
			info.Types = append(info.Types, typeInfo)
		}
		return true
	})

	return info, nil
}

// groupByStructure groups files by their structural patterns
func groupByStructure(files []patterns.FileInfo) map[patterns.PatternType][]patterns.FileInfo {
	groups := make(map[patterns.PatternType][]patterns.FileInfo)

	for _, file := range files {
		patternType := inferPatternType(file)
		groups[patternType] = append(groups[patternType], file)
	}

	return groups
}

// inferPatternType determines what kind of pattern a file represents
func inferPatternType(file patterns.FileInfo) patterns.PatternType {
	// Check file path and package
	if strings.Contains(file.Path, "/handlers/") || strings.Contains(file.Path, "/controllers/") {
		return patterns.PatternHTTPHandler
	}
	if strings.Contains(file.Path, "/services/") {
		return patterns.PatternService
	}
	if strings.Contains(file.Path, "/repository/") || strings.Contains(file.Path, "/repositories/") {
		return patterns.PatternRepository
	}
	if strings.Contains(file.Path, "/middleware/") {
		return patterns.PatternMiddleware
	}
	if strings.Contains(file.Path, "/models/") {
		return patterns.PatternModel
	}

	// Check function signatures
	for _, fn := range file.Functions {
		// HTTP handler pattern: func XxxHandler(w http.ResponseWriter, r *http.Request)
		if strings.HasSuffix(fn.Name, "Handler") {
			return patterns.PatternHTTPHandler
		}
		// Middleware pattern: func XxxMiddleware(next http.Handler) http.Handler
		if strings.HasSuffix(fn.Name, "Middleware") {
			return patterns.PatternMiddleware
		}
	}

	// Check types
	for _, typ := range file.Types {
		if strings.HasSuffix(typ.Name, "Service") {
			return patterns.PatternService
		}
		if strings.HasSuffix(typ.Name, "Repository") {
			return patterns.PatternRepository
		}
	}

	// Infer pattern type from package name (supports internal/pkg organization)
	pkgPatternType := inferFromPackageName(file.Package)
	if pkgPatternType != patterns.PatternUtil {
		return pkgPatternType
	}

	return patterns.PatternUtil
}

// inferFromPackageName infers pattern type from the package name
func inferFromPackageName(pkgName string) patterns.PatternType {
	switch pkgName {
	case "analyzer", "parser", "scanner", "lexer":
		return patterns.PatternService // Analyzers are service-like
	case "config", "configuration", "settings":
		return patterns.PatternUtil // Config is utility
	case "detector", "finder", "locator", "discovery":
		return patterns.PatternService // Detectors are service-like
	case "matcher", "comparator", "validator":
		return patterns.PatternService // Matchers are service-like
	case "reporter", "writer", "output", "formatter":
		return patterns.PatternService // Reporters are service-like
	case "handler", "handlers", "controller", "controllers":
		return patterns.PatternHTTPHandler
	case "service", "services":
		return patterns.PatternService
	case "repository", "repositories", "repo", "repos", "store", "storage":
		return patterns.PatternRepository
	case "middleware", "middlewares":
		return patterns.PatternMiddleware
	case "model", "models", "entity", "entities", "domain":
		return patterns.PatternModel
	default:
		return patterns.PatternUtil
	}
}

// extractPattern creates a pattern from a group of similar files
func extractPattern(patternType patterns.PatternType, group []patterns.FileInfo) patterns.Pattern {
	pattern := patterns.Pattern{
		ID:         generatePatternID(patternType),
		Name:       string(patternType),
		Type:       patternType,
		Confidence: calculateConfidence(group),
		SeenCount:  len(group),
		Version:    "1.0",
	}

	// Set detection rules based on pattern type
	switch patternType {
	case patterns.PatternHTTPHandler:
		pattern.Detection = patterns.DetectionRule{
			FilePattern: "*_handler.go",
			FuncPattern: "func.*Handler.*http\\.ResponseWriter",
			PackagePath: "*/handlers",
		}
	case patterns.PatternService:
		pattern.Detection = patterns.DetectionRule{
			FilePattern:   "*_service.go",
			StructPattern: "type.*Service.*struct",
			PackagePath:   "*/services",
		}
	case patterns.PatternRepository:
		pattern.Detection = patterns.DetectionRule{
			FilePattern:   "*_repo.go",
			StructPattern: "type.*Repository.*interface",
			PackagePath:   "*/repository",
		}
	}

	// Extract common structure elements
	pattern.Structure = extractCommonStructure(group)

	return pattern
}

// extractCommonStructure finds elements common across all files
func extractCommonStructure(group []patterns.FileInfo) patterns.CodeStructure {
	structure := patterns.CodeStructure{
		Elements: []patterns.StructureElement{},
		Ordering: []string{},
		Required: []string{},
		Optional: []string{},
	}

	// Find common imports
	importCounts := make(map[string]int)
	for _, file := range group {
		for _, imp := range file.Imports {
			importCounts[imp]++
		}
	}

	// Imports present in >80% of files are "required"
	threshold := int(float64(len(group)) * 0.8)
	for imp, count := range importCounts {
		if count >= threshold {
			structure.Required = append(structure.Required, imp)
			structure.Elements = append(structure.Elements, patterns.StructureElement{
				Name:    imp,
				Type:    patterns.ElementImport,
				Pattern: regexp.QuoteMeta(imp),
			})
		}
	}

	return structure
}

// generatePatternID creates a unique ID for a pattern
func generatePatternID(patternType patterns.PatternType) string {
	return fmt.Sprintf("%s_pattern", patternType)
}

// calculateConfidence returns confidence score for a pattern
func calculateConfidence(group []patterns.FileInfo) float64 {
	// More examples = higher confidence
	// 3 files = 0.6, 5 files = 0.75, 10+ files = 0.9
	count := float64(len(group))
	if count >= 10 {
		return 0.9
	}
	if count >= 5 {
		return 0.75
	}
	if count >= 3 {
		return 0.6
	}
	return 0.5
}
