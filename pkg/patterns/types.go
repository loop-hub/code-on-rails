package patterns

import "time"

// Pattern represents a detected code pattern in the codebase
type Pattern struct {
	ID                string           `yaml:"id"`
	Name              string           `yaml:"name"`
	Type              PatternType      `yaml:"type"`
	Version           string           `yaml:"version"`
	Detection         DetectionRule    `yaml:"detection"`
	Structure         CodeStructure    `yaml:"structure"`
	AnnotatedGolden   []GoldenExample  `yaml:"annotated_golden,omitempty"`
	ConfigBlessed     []BlessedExample `yaml:"config_blessed,omitempty"`
	Discovered        []Example        `yaml:"discovered,omitempty"`
	AntiPatterns      []AntiPattern    `yaml:"anti_patterns,omitempty"`
	Confidence        float64          `yaml:"confidence"`
	SeenCount         int              `yaml:"seen_count"`
}

// GoldenExample represents an annotated golden example
type GoldenExample struct {
	Path         string    `yaml:"path"`
	Function     string    `yaml:"function,omitempty"`
	Pattern      string    `yaml:"pattern"`
	Version      string    `yaml:"version,omitempty"`
	BlessedBy    string    `yaml:"blessed_by"`
	BlessedDate  time.Time `yaml:"blessed_date"`
	Reason       string    `yaml:"reason"`
	QualityScore int       `yaml:"quality_score,omitempty"`
	Weight       float64   `yaml:"weight"`
}

// BlessedExample represents a config-blessed example
type BlessedExample struct {
	Path        string    `yaml:"path"`
	Function    string    `yaml:"function,omitempty"`
	BlessedBy   string    `yaml:"blessed_by"`
	BlessedDate time.Time `yaml:"blessed_date"`
	Reason      string    `yaml:"reason"`
	Weight      float64   `yaml:"weight"`
}

// Example represents an auto-discovered example
type Example struct {
	Path            string  `yaml:"path"`
	SimilarityScore float64 `yaml:"similarity_score,omitempty"`
	SeenCount       int     `yaml:"seen_count,omitempty"`
	Weight          float64 `yaml:"weight"`
}

// AntiPattern represents a pattern to avoid
type AntiPattern struct {
	Path           string     `yaml:"path"`
	Function       string     `yaml:"function,omitempty"`
	Pattern        string     `yaml:"pattern"`
	Reason         string     `yaml:"reason"`
	Deprecated     *time.Time `yaml:"deprecated,omitempty"`
	MigrationGuide string     `yaml:"migration_guide,omitempty"`
}

// PatternType represents the category of pattern
type PatternType string

const (
	// Go patterns
	PatternHTTPHandler PatternType = "http_handler"
	PatternService     PatternType = "service"
	PatternRepository  PatternType = "repository"
	PatternMiddleware  PatternType = "middleware"
	PatternModel       PatternType = "model"
	PatternUtil        PatternType = "util"

	// TypeScript/React patterns
	PatternComponent     PatternType = "component"      // React functional/class components
	PatternHook          PatternType = "hook"           // Custom React hooks
	PatternContext       PatternType = "context"        // React context providers
	PatternPage          PatternType = "page"           // Next.js/React Router pages
	PatternAPI           PatternType = "api"            // API route handlers (Next.js, Express)
	PatternStore         PatternType = "store"          // State management (Redux, Zustand)
	PatternTypeDefinition PatternType = "types"          // TypeScript type definitions
	PatternTest          PatternType = "test"           // Test files
	PatternStorybook     PatternType = "storybook"      // Storybook stories
	PatternStyled        PatternType = "styled"         // Styled components
)

// DetectionRule defines how to detect this pattern
type DetectionRule struct {
	FilePattern   string `yaml:"file_pattern"`
	FuncPattern   string `yaml:"func_pattern"`
	StructPattern string `yaml:"struct_pattern"`
	PackagePath   string `yaml:"package_path"`
}

// CodeStructure describes the expected structure
type CodeStructure struct {
	Elements []StructureElement `yaml:"elements"`
	Ordering []string           `yaml:"ordering"`
	Required []string           `yaml:"required"`
	Optional []string           `yaml:"optional"`
}

// StructureElement represents a component of the code structure
type StructureElement struct {
	Name     string      `yaml:"name"`
	Type     ElementType `yaml:"type"`
	Pattern  string      `yaml:"pattern"`
	Examples []string    `yaml:"examples"`
}

// ElementType categorizes structural elements
type ElementType string

const (
	// Common elements
	ElementImport      ElementType = "import"
	ElementTypeDecl    ElementType = "type"
	ElementFunction    ElementType = "function"
	ElementMethod      ElementType = "method"
	ElementErrorHandle ElementType = "error_handling"
	ElementValidation  ElementType = "validation"
	ElementTransaction ElementType = "transaction"

	// TypeScript/React elements
	ElementInterface    ElementType = "interface"     // TypeScript interface
	ElementTypeAlias    ElementType = "type_alias"    // TypeScript type alias
	ElementEnum         ElementType = "enum"          // TypeScript enum
	ElementJSXElement   ElementType = "jsx_element"   // JSX/TSX element
	ElementHookCall     ElementType = "hook_call"     // React hook usage (useState, useEffect, etc.)
	ElementProp         ElementType = "prop"          // Component prop
	ElementState        ElementType = "state"         // Component state
	ElementEffect       ElementType = "effect"        // useEffect/side effects
	ElementExport       ElementType = "export"        // Export statement
	ElementDefaultExport ElementType = "default_export" // Default export
)

// PatternMatch represents how well code matches a pattern
type PatternMatch struct {
	Pattern        *Pattern
	FilePath       string
	Score          float64
	WeightedScore  float64
	MatchType      string // "annotated_golden", "config_blessed", "discovered"
	GoldenRef      *GoldenExample
	BlessedRef     *BlessedExample
	DiscoveredRef  *Example
	Deviations     []Deviation
	AutoApprove    bool
}

// Deviation describes how code differs from pattern
type Deviation struct {
	Type        DeviationType
	Element     string
	Expected    string
	Actual      string
	Severity    Severity
	Suggestion  string
	LineNumber  int
}

// DeviationType categorizes deviations
type DeviationType string

const (
	DeviationMissing   DeviationType = "missing"
	DeviationDifferent DeviationType = "different"
	DeviationNovel     DeviationType = "novel"
)

// Severity levels for deviations
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// FileInfo represents a parsed file
type FileInfo struct {
	Path      string
	Package   string
	Imports   []string
	Functions []FunctionInfo
	Types     []TypeInfo
}

// FunctionInfo represents a function or method
type FunctionInfo struct {
	Name       string
	Receiver   string
	Parameters []string
	Returns    []string
	Body       string
}

// TypeInfo represents a type definition
type TypeInfo struct {
	Name   string
	Kind   string // struct, interface, etc.
	Fields []string
}
