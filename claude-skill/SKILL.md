---
name: code-on-rails
description: Automatically enforce codebase patterns when writing code. Use this skill when generating code in a project that has a .code-on-rails.yml configuration file. The skill reads existing patterns and ensures all generated code follows established conventions, preventing pattern drift and reducing code review overhead.
---

# Code on Rails Integration

This skill enables Claude Code to automatically follow your codebase patterns by reading the `.code-on-rails.yml` configuration file.

## Automatic Feedback Loop (Priority Check)

**IMPORTANT**: Before starting any work, ALWAYS check for a feedback file:

```bash
cat .code-on-rails-feedback.json 2>/dev/null || echo "No feedback file"
```

If this file exists, it contains AI-readable feedback from the CI workflow about pattern deviations that need fixing. **Read it first and fix the issues before doing anything else.**

### Reading the Feedback File

The feedback file structure:

```json
{
  "summary": {
    "total_files": 5,
    "needs_fixes": 2,
    "auto_approved": 3
  },
  "files_to_fix": [
    {
      "file_path": "src/handlers/payment.go",
      "pattern_type": "http_handler",
      "issues": [
        {
          "element": "error_handling",
          "expected": "errors.Wrap",
          "suggestion": "Use errors.Wrap for error context"
        }
      ],
      "reference_file": "src/handlers/user_handler.go"
    }
  ],
  "instructions": "..."
}
```

### Fixing Feedback Issues

1. **Read the reference file** for each file that needs fixes:
   ```bash
   view <reference_file>
   ```

2. **Read the file that needs fixes**:
   ```bash
   view <file_path>
   ```

3. **Apply each fix** based on the suggestion, matching the reference pattern

4. **Verify the fix**:
   ```bash
   cr check <file_path>
   ```

5. **Delete the feedback file** once all issues are resolved:
   ```bash
   rm .code-on-rails-feedback.json
   ```

6. **Commit the fixes**:
   ```bash
   git add -A && git commit -m "fix: resolve Code on Rails pattern deviations"
   git push
   ```

---

## When to Use

Automatically triggers when:
- A `.code-on-rails.yml` file exists in the project
- A `.code-on-rails-feedback.json` file exists (fix issues first!)
- User requests code generation (handlers, services, models, etc.)
- User asks to "follow our patterns" or "match existing code"

## How It Works

### 1. Check for Feedback (Always First!)

```bash
cat .code-on-rails-feedback.json 2>/dev/null
```

If feedback exists, fix those issues before proceeding.

### 2. Read Configuration

Before writing any code, read the pattern definitions:

```bash
cat .code-on-rails.yml
```

This shows:
- **Pattern types**: HTTP handlers, services, repositories, etc.
- **Reference implementations**: Canonical examples to follow
- **Required elements**: Imports, error handling, validation patterns
- **Detection rules**: File naming conventions

### 3. Study Reference Files

For the pattern type you're creating, examine the reference implementations:

```bash
# If creating a handler, view the reference handler
view internal/handlers/user_handler.go

# If creating a service, view the reference service
view internal/services/user_service.go
```

Look for:
- Import patterns
- Function signatures
- Error handling approach
- Validation strategy
- Naming conventions
- Code structure and flow

### 4. Generate Code Following Patterns

When writing code:

**Match the structure** of reference implementations:
- Same import organization
- Same error handling patterns
- Same validation approach
- Same function/method ordering
- Same naming conventions

**Include required elements** from the pattern:
- All required imports (check `structure.required` in config)
- Expected function signatures
- Error wrapping style
- Transaction patterns (for database code)

**Follow the example flow**:
If reference handler does: validate -> service call -> response,
Your handler should follow the same flow.

### 5. Self-Check

Before finishing, verify your code:

```bash
# Check if the file matches patterns
cr check path/to/new_file.go
```

If the similarity score is below 95%, review deviations and adjust.

## Pattern Type Guidelines

### HTTP Handlers

When creating handlers, follow this structure (based on reference):

```go
// Reference: internal/handlers/user_handler.go

func ProductHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Validate input (using validator from config)
    var req ProductRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        // Use error handling pattern from reference
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 2. Call service layer
    result, err := productService.GetProduct(req.ID)
    if err != nil {
        // Match error wrapping style
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 3. Return response
    json.NewEncoder(w).Encode(result)
}
```

Key points:
- Match the reference's error handling style
- Use the same validation library
- Follow the same 3-step flow
- Use the same response encoding method

### Services

When creating services, follow this structure:

```go
// Reference: internal/services/user_service.go

type ProductService struct {
    repo ProductRepository
}

func NewProductService(repo ProductRepository) *ProductService {
    return &ProductService{repo: repo}
}

func (s *ProductService) GetProduct(id string) (*Product, error) {
    // Match error handling from reference
    product, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errors.Wrap(err, "failed to find product")
    }
    return product, nil
}
```

Key points:
- Dependency injection via constructor
- Error wrapping style (errors.Wrap vs fmt.Errorf)
- Repository pattern usage
- Method naming conventions

### Repositories

When creating repositories:

```go
// Reference: internal/repository/user_repo.go

type ProductRepository interface {
    FindByID(id string) (*Product, error)
    Create(p *Product) error
}

type productRepo struct {
    db *gorm.DB
}

func (r *productRepo) FindByID(id string) (*Product, error) {
    var product Product

    // Match transaction pattern if required
    if err := r.db.First(&product, "id = ?", id).Error; err != nil {
        return nil, err
    }

    return &product, nil
}
```

Key points:
- Interface definition
- Database abstraction
- Transaction handling
- Query patterns

## Reading the Config

The `.code-on-rails.yml` structure:

```yaml
patterns:
  - id: http_handler_pattern
    name: HTTP Handler
    examples:
      - "internal/handlers/user_handler.go"  # Study this file
    structure:
      required:
        - "net/http"              # Must import this
        - "encoding/json"         # Must import this
```

For each pattern:
1. Note the **examples** - these are your templates
2. Check **required** elements - must include these
3. Read **detection** rules - for naming conventions

## Example Workflow

User: "Create a product search handler"

1. **Check for feedback first**:
   ```bash
   cat .code-on-rails-feedback.json 2>/dev/null
   ```
   -> If exists, fix those issues first!

2. **Read config**:
   ```bash
   cat .code-on-rails.yml
   ```
   -> Identifies `http_handler_pattern`

3. **Study reference**:
   ```bash
   view internal/handlers/user_handler.go
   ```
   -> Notes: uses validator.v10, errors.Wrap, 3-step flow

4. **Generate code** matching the pattern:
   - Same imports
   - Same error handling
   - Same validation approach
   - Same structure

5. **Verify**:
   ```bash
   cr check internal/handlers/product_search_handler.go
   ```
   -> Should show 95%+ match

## Anti-Patterns to Avoid

- **Don't** invent new patterns when one exists
- **Don't** skip required imports
- **Don't** reorder the flow arbitrarily
- **Don't** ignore the feedback file

## Configuration Not Found

If `.code-on-rails.yml` doesn't exist, suggest running:

```bash
cr init
```

This will scan the codebase and generate the configuration automatically.

## Benefits

By following patterns automatically:
- **95%+ auto-approval rate** in code review
- **Consistent codebase** across the team
- **Faster reviews** - reviewers focus on business logic
- **Less rework** - code matches expectations first time
- **Better AI adoption** - confidence in AI-generated code

---

**Remember**: The goal is consistency, not perfection. Follow the existing pattern even if you think there's a "better" way. Pattern evolution happens through team discussion, not individual deviation.
