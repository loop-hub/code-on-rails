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
	switch a.Language {
	case "go":
		return a.extractGoPatterns(rootPath)
	case "typescript", "ts", "javascript", "js", "react":
		return a.extractTypeScriptPatterns(rootPath)
	default:
		return nil, fmt.Errorf("unsupported language: %s", a.Language)
	}
}

// extractGoPatterns extracts patterns from Go codebases
func (a *Analyzer) extractGoPatterns(rootPath string) ([]patterns.Pattern, error) {

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

// extractTypeScriptPatterns extracts patterns from TypeScript/JavaScript codebases
func (a *Analyzer) extractTypeScriptPatterns(rootPath string) ([]patterns.Pattern, error) {
	// Find all TypeScript/JavaScript files
	files, err := findTypeScriptFiles(rootPath)
	if err != nil {
		return nil, err
	}

	// Parse all files
	fileInfos := make([]patterns.FileInfo, 0, len(files))
	for _, file := range files {
		info, err := parseTypeScriptFile(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, *info)
	}

	// Group files by pattern type
	groups := groupTypeScriptByPattern(fileInfos)

	// Extract patterns from groups
	extractedPatterns := []patterns.Pattern{}
	for patternType, group := range groups {
		if len(group) < 2 {
			continue // Need at least 2 examples
		}

		pattern := extractTypeScriptPattern(patternType, group)
		pattern.Discovered = make([]patterns.Example, 0, len(group))
		for _, file := range group {
			pattern.Discovered = append(pattern.Discovered, patterns.Example{
				Path:            file.Path,
				SimilarityScore: 0.9,
				Weight:          1.0,
			})
		}

		extractedPatterns = append(extractedPatterns, pattern)
	}

	return extractedPatterns, nil
}

// findTypeScriptFiles recursively finds all TypeScript/JavaScript files
func findTypeScriptFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip node_modules and common non-source directories
			if info.Name() == "node_modules" || info.Name() == "dist" ||
				info.Name() == "build" || info.Name() == ".next" {
				return filepath.SkipDir
			}
			return nil
		}

		// Include TypeScript and JavaScript files
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
			// Skip test files
			name := strings.ToLower(filepath.Base(path))
			if !strings.Contains(name, ".test.") && !strings.Contains(name, ".spec.") &&
				!strings.Contains(path, "__tests__") && !strings.Contains(path, "__mocks__") {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

// parseTypeScriptFile parses a TypeScript/JavaScript file using text analysis
func parseTypeScriptFile(filePath string) (*patterns.FileInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	code := string(content)
	lines := strings.Split(code, "\n")

	info := &patterns.FileInfo{
		Path:      filePath,
		Package:   filepath.Base(filepath.Dir(filePath)),
		Imports:   extractTypeScriptImports(code),
		Functions: extractTypeScriptFunctions(lines),
		Types:     extractTypeScriptTypes(lines),
	}

	return info, nil
}

// extractTypeScriptImports extracts import statements from TypeScript/JavaScript
func extractTypeScriptImports(code string) []string {
	imports := []string{}
	importRegex := regexp.MustCompile(`import\s+(?:{[^}]+}|[\w\s,*]+)\s+from\s+['"]([^'"]+)['"]`)
	matches := importRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}
	return imports
}

// extractTypeScriptFunctions extracts function declarations
func extractTypeScriptFunctions(lines []string) []patterns.FunctionInfo {
	functions := []patterns.FunctionInfo{}

	// Patterns for function declarations
	funcPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+(\w+)`),
		regexp.MustCompile(`(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s+)?\(`),
		regexp.MustCompile(`(?:export\s+)?const\s+(\w+)\s*:\s*\w+\s*=\s*(?:async\s+)?\(`),
		regexp.MustCompile(`(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s+)?\([^)]*\)\s*=>`),
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, pattern := range funcPatterns {
			if matches := pattern.FindStringSubmatch(trimmed); len(matches) > 1 {
				functions = append(functions, patterns.FunctionInfo{
					Name: matches[1],
				})
				break
			}
		}
	}

	return functions
}

// extractTypeScriptTypes extracts type and interface declarations
func extractTypeScriptTypes(lines []string) []patterns.TypeInfo {
	types := []patterns.TypeInfo{}

	typePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:export\s+)?interface\s+(\w+)`),
		regexp.MustCompile(`(?:export\s+)?type\s+(\w+)\s*=`),
		regexp.MustCompile(`(?:export\s+)?class\s+(\w+)`),
		regexp.MustCompile(`(?:export\s+)?enum\s+(\w+)`),
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for i, pattern := range typePatterns {
			if matches := pattern.FindStringSubmatch(trimmed); len(matches) > 1 {
				kind := []string{"interface", "type", "class", "enum"}[i]
				types = append(types, patterns.TypeInfo{
					Name: matches[1],
					Kind: kind,
				})
				break
			}
		}
	}

	return types
}

// groupTypeScriptByPattern groups files by their pattern type
func groupTypeScriptByPattern(files []patterns.FileInfo) map[patterns.PatternType][]patterns.FileInfo {
	groups := make(map[patterns.PatternType][]patterns.FileInfo)

	for _, file := range files {
		patternType := inferTypeScriptPatternType(file)
		groups[patternType] = append(groups[patternType], file)
	}

	return groups
}

// inferTypeScriptPatternType determines the pattern type for a TypeScript file
func inferTypeScriptPatternType(file patterns.FileInfo) patterns.PatternType {
	fileName := strings.ToLower(filepath.Base(file.Path))
	filePath := strings.ToLower(file.Path)

	// Check file name patterns
	if strings.HasSuffix(fileName, ".component.tsx") || strings.HasSuffix(fileName, ".component.jsx") {
		return patterns.PatternComponent
	}
	if strings.HasPrefix(fileName, "use") && (strings.HasSuffix(fileName, ".ts") || strings.HasSuffix(fileName, ".tsx")) {
		return patterns.PatternHook
	}
	if strings.Contains(fileName, ".hook.") || strings.Contains(fileName, "hooks") {
		return patterns.PatternHook
	}
	if strings.Contains(fileName, ".context.") || strings.Contains(filePath, "/context/") {
		return patterns.PatternContext
	}
	if strings.Contains(filePath, "/pages/") || strings.Contains(filePath, "/app/") {
		return patterns.PatternPage
	}
	if strings.Contains(filePath, "/api/") || strings.Contains(fileName, ".api.") {
		return patterns.PatternAPI
	}
	if strings.Contains(fileName, ".store.") || strings.Contains(filePath, "/store/") ||
		strings.Contains(filePath, "/redux/") || strings.Contains(filePath, "/zustand/") {
		return patterns.PatternStore
	}
	if strings.HasSuffix(fileName, ".types.ts") || strings.HasSuffix(fileName, ".d.ts") ||
		strings.Contains(filePath, "/types/") {
		return patterns.PatternTypeDefinition
	}
	if strings.Contains(fileName, ".stories.") || strings.Contains(filePath, "/stories/") {
		return patterns.PatternStorybook
	}
	if strings.Contains(fileName, ".styled.") || strings.Contains(fileName, ".styles.") {
		return patterns.PatternStyled
	}
	if strings.Contains(fileName, ".service.") || strings.Contains(filePath, "/services/") {
		return patterns.PatternService
	}

	// Check for React components by content
	for _, imp := range file.Imports {
		if imp == "react" || strings.HasPrefix(imp, "react/") {
			// Check for component patterns in functions
			for _, fn := range file.Functions {
				if strings.HasPrefix(fn.Name, "use") {
					return patterns.PatternHook
				}
			}
			return patterns.PatternComponent
		}
	}

	return patterns.PatternUtil
}

// extractTypeScriptPattern creates a pattern from grouped TypeScript files
func extractTypeScriptPattern(patternType patterns.PatternType, files []patterns.FileInfo) patterns.Pattern {
	// Collect common imports
	importCounts := make(map[string]int)
	for _, file := range files {
		for _, imp := range file.Imports {
			importCounts[imp]++
		}
	}

	// Find common imports (present in >50% of files)
	threshold := len(files) / 2
	if threshold < 1 {
		threshold = 1
	}

	commonImports := []string{}
	for imp, count := range importCounts {
		if count >= threshold {
			commonImports = append(commonImports, imp)
		}
	}

	// Build pattern structure
	elements := make([]patterns.StructureElement, 0, len(commonImports))
	for _, imp := range commonImports {
		elements = append(elements, patterns.StructureElement{
			Name:    imp,
			Type:    patterns.ElementImport,
			Pattern: regexp.QuoteMeta(imp),
		})
	}

	return patterns.Pattern{
		ID:       string(patternType) + "_pattern",
		Name:     string(patternType),
		Type:     patternType,
		Version:  "1.0",
		Structure: patterns.CodeStructure{
			Elements: elements,
			Required: commonImports,
		},
		Confidence: 0.8,
		SeenCount:  len(files),
	}
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
