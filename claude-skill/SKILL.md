---
name: code-on-rails
description: Automatically enforce codebase patterns when writing code. Use this skill when generating code in a project that has a .code-on-rails.yml configuration file. The skill reads existing patterns and ensures all generated code follows established conventions, preventing pattern drift and reducing code review overhead.
---

# Code on Rails Integration

This skill enables Claude Code to automatically follow your codebase patterns by reading the `.code-on-rails.yml` configuration file.

## When to Use

Automatically triggers when:
- A `.code-on-rails.yml` file exists in the project
- User requests code generation (handlers, services, models, etc.)
- User asks to "follow our patterns" or "match existing code"

## How It Works

### 1. Read Configuration

Before writing any code, read the pattern definitions:

```bash
view .code-on-rails.yml
```

This shows:
- **Pattern types**: HTTP handlers, services, repositories, etc.
- **Reference implementations**: Canonical examples to follow
- **Required elements**: Imports, error handling, validation patterns
- **Detection rules**: File naming conventions

### 2. Study Reference Files

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

### 3. Generate Code Following Patterns

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
If reference handler does: validate → service call → response,
Your handler should follow the same flow.

### 4. Self-Check

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

1. **Read config**:
   ```bash
   view .code-on-rails.yml
   ```
   → Identifies `http_handler_pattern`

2. **Study reference**:
   ```bash
   view internal/handlers/user_handler.go
   ```
   → Notes: uses validator.v10, errors.Wrap, 3-step flow

3. **Generate code** matching the pattern:
   - Same imports
   - Same error handling
   - Same validation approach
   - Same structure

4. **Verify**:
   ```bash
   cr check internal/handlers/product_search_handler.go
   ```
   → Should show 95%+ match

5. **Commit with tag**:
   ```bash
   git commit -m "[claude] Add product search handler"
   ```

## Anti-Patterns to Avoid

❌ **Don't** invent new patterns when one exists:
```go
// BAD: New error handling when reference uses errors.Wrap
return fmt.Errorf("error: %v", err)

// GOOD: Match reference pattern
return errors.Wrap(err, "failed to process")
```

❌ **Don't** skip required imports:
```go
// BAD: Missing required validation
func Handler(w http.ResponseWriter, r *http.Request) {
    // No validation...
}

// GOOD: Include validation like reference
func Handler(w http.ResponseWriter, r *http.Request) {
    if err := validator.Validate(req); err != nil {
        // ...
    }
}
```

❌ **Don't** reorder the flow arbitrarily:
```go
// BAD: Different order than reference
func Handler(...) {
    result := service.Call()
    validate()
    return result
}

// GOOD: Same flow as reference
func Handler(...) {
    validate()
    result := service.Call()
    return result
}
```

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

## Tips

1. **Always read the reference file** - Don't guess the pattern
2. **Match exactly first time** - Easier than fixing later
3. **Use the same libraries** - Don't introduce new dependencies
4. **Follow the flow** - Structure matters, not just syntax
5. **Self-check with cr** - Verify before committing

## Advanced: Multiple Patterns

If creating code that spans multiple patterns:

```bash
# Creating a feature with handler + service + repository

# 1. Read all relevant patterns
view .code-on-rails.yml

# 2. Study each reference
view internal/handlers/user_handler.go
view internal/services/user_service.go  
view internal/repository/user_repo.go

# 3. Create each file following its pattern
# Handler follows handler pattern
# Service follows service pattern
# Repository follows repository pattern

# 4. Ensure they integrate like references do
# Study how user_handler calls user_service
# Study how user_service calls user_repo
```

## Updating Patterns

If you notice the pattern should change:

1. Implement the better approach
2. Update `.code-on-rails.yml` to reflect it
3. Update reference implementations
4. Or suggest pattern update to team

Don't silently deviate - either follow the pattern or propose changing it.

---

**Remember**: The goal is consistency, not perfection. Follow the existing pattern even if you think there's a "better" way. Pattern evolution happens through team discussion, not individual deviation.
