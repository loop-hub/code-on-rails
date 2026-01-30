# Code on Rails

**Pattern enforcement for AI-generated code with hybrid annotation system.**

Code on Rails (`cr`) learns your codebase patterns and ensures AI-generated code matches your team's standards. Works with Claude Code, Copilot, Cursor, and any AI tool.

## The Problem

You're shipping code faster with AI, but code reviews are bottlenecked:
- Each microservice is developing its own patterns
- Reviewers spend time checking "does this match our style?" instead of business logic
- AI tools don't know your existing patterns
- Inconsistency compounds across dozens of services

## The Solution

Code on Rails automatically:
1. **Learns patterns** from your existing codebase (zero config bootstrap)
2. **Validates new code** against those patterns before review
3. **Auto-approves** code that matches (95%+ similarity)
4. **Flags deviations** with specific suggestions
5. **Learns continuously** from merged code

## Quick Start

```bash
# Install
go install github.com/loop-hub/code-on-rails/cmd/cr@latest

# Option 1: Quick auto-discovery (large codebase)
cd your-microservice/
cr init  # Discovers patterns automatically

# Option 2: Golden annotation (greenfield project)
# Add to your best code:
# @code-on-rails: golden-example
# @pattern: http_handler
# @author: @your-name
# @blessed: 2025-01-30
cr init  # Detects annotations + discovers patterns

# Check AI-generated code
cr check

# Or check specific files
cr check internal/handlers/product_handler.go
```

## The Hybrid Approach

### For Large Codebases
```bash
# Week 1: Auto-discover everything
cr init --recursive

# Week 2: Bless top patterns
cr bless internal/handlers/user_handler.go --reason "Best practices"

# Ongoing: Annotate critical examples (1-2 per month)
# Add annotations to your absolute best code
```

**Result:** 100% coverage from day 1, progressive quality improvement

### For Greenfield Projects
```bash
# Day 1: Engineer writes golden example with annotation
cr init

# Day 2+: Claude generates matching code
# Claude reads annotation, follows pattern, auto-approved at 98%+
```

**Result:** Quality patterns from day 1, consistent codebase

## How It Works

### 1. Bootstrap (One Time)

```bash
$ cr init

Scanning codebase...
→ Discovered 247 Go files
→ Identified patterns:
  • 23 HTTP handlers
  • 15 database models  
  • 8 middleware functions
  • 12 service layer structs
→ Generated .code-on-rails.yml
✓ Ready to use!
```

### 2. Pre-commit Check

```bash
$ cr check

Analyzing AI-generated code...

✓ internal/handlers/product_handler.go
  Pattern: HTTP Handler (98% match)
  Reference: internal/handlers/user_handler.go
  Auto-approved

⚠ internal/services/product_service.go  
  Pattern: Service Layer (87% match)
  Deviations:
    - Different error handling (expected errors.Wrap, found fmt.Errorf)
    - Missing transaction wrapper for database calls
  Suggestion: See internal/services/user_service.go:45-52
  Needs review

Summary:
  ✓ 1 file auto-approved (98 lines)
  ⚠ 1 file needs review (45 lines)
  
Estimated review time saved: 12 minutes
```

### 3. GitHub Action (Automated)

```yaml
# .github/workflows/code-on-rails.yml
name: Code on Rails
on: [pull_request]

jobs:
  pattern-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: code-on-rails/action@v1
        with:
          auto-approve-threshold: 95
```

Posts a comment on your PR:
```
✅ Auto-approved (3 files):
  - internal/handlers/product_search_handler.go (98% match)
  - internal/services/product_service.go (96% match)
  - internal/models/product.go (97% match)

⚠️ Needs review (1 file):
  - internal/repository/product_repo.go
    Reason: New database query pattern detected
    Novel code: lines 45-67 (custom JOIN not seen before)

📊 Stats: 156 lines auto-approved, 23 lines flagged
```

## Configuration

Each microservice has its own `.code-on-rails.yml` with hybrid pattern support:

```yaml
version: "1.0"
language: go
ai_source: any  # or "claude", "copilot", "cursor"

patterns:
  - id: http_handler_pattern
    name: HTTP Handler
    version: "1.0"
    
    # 🏆 Tier 1: Annotated Golden (highest priority, weight 2.0x)
    annotated_golden:
      - path: internal/handlers/user_handler.go
        function: UserHandler
        blessed_by: "@senior-dev-alice"
        blessed_date: 2025-01-30
        reason: "Template for all handlers. Clean validation, error handling."
        quality_score: 95
        weight: 2.0
    
    # ⭐ Tier 2: Config Blessed (high priority, weight 1.5x)
    config_blessed:
      - path: internal/handlers/auth_handler.go
        blessed_by: "config"
        blessed_date: 2025-01-30
        reason: "Security best practices"
        weight: 1.5
    
    # 📊 Tier 3: Auto-Discovered (normal priority, weight 1.0x)
    discovered:
      - path: internal/handlers/product_handler.go
        similarity_score: 0.95
        weight: 1.0
      - path: internal/handlers/order_handler.go
        similarity_score: 0.93
        weight: 1.0
    
    # Detection rules
    detection:
      file_pattern: "*_handler.go"
      func_pattern: "func.*Handler.*http\\.ResponseWriter"
    
    # Confidence metrics
    confidence: 0.95
    seen_count: 23

settings:
  auto_approve_threshold: 95
  prefer_annotated: true  # Always prefer annotated over discovered
  learn_on_merge: true
```

### Three Ways to Mark Patterns

**1. Annotate in Code (Best for critical patterns)**
```go
// @code-on-rails: golden-example
// @pattern: http_handler
// @author: @your-name
// @blessed: 2025-01-30
func YourHandler(...) { }
```

**2. Bless via Config (Good for important patterns)**
```bash
cr bless internal/handlers/auth_handler.go --reason "Security best practices"
```

**3. Auto-Discover (Default for everything else)**
```bash
cr init  # Automatically finds patterns
```

## 🎯 Claude Skill - The Feedback Loop!

Install the Code on Rails skill to make Claude Code automatically follow your patterns BEFORE writing code:

```bash
# Install the skill
cp -r claude-skill ~/.claude/skills/code-on-rails
```

Now when you ask Claude Code to generate code:
1. Claude automatically reads `.code-on-rails.yml`
2. Studies your reference implementations
3. Generates code matching your patterns
4. Self-validates with `cr check`

**Result:** 95%+ auto-approval rate instead of manual pattern fixes!

See [`claude-skill/INSTALL.md`](claude-skill/INSTALL.md) for full instructions.

### Before vs After

**Before (without skill):**
```
You: Create a product handler
Claude: [generic handler]
cr check: 67% match ❌
→ Needs rework
```

**After (with skill):**
```
You: Create a product handler  
Claude: [reads patterns, studies examples, generates matching code]
cr check: 98% match ✅
→ Auto-approved!
```

## Detecting AI-Generated Code

Code on Rails detects AI code through:

1. **Commit messages**: `git commit -m "[ai:claude] Add product endpoint"`
2. **Git notes**: `git notes add -m "ai-tool: copilot"`
3. **Heuristics**: Patterns typical of AI (comprehensive comments, complete error handling, etc.)

Configure in `.code-on-rails.yml`:
```yaml
detection:
  method: "commit_message"  # or "git_notes", "heuristic", "all"
  commit_prefixes: ["[ai", "[claude", "[copilot"]
```

## Features

- **🎯 Hybrid System**: Auto-discovery + annotations + config blessing
- **🏆 Golden Examples**: Annotate your best code inline with comments
- **⚡ Weighted Matching**: Annotated (2.0x) > Blessed (1.5x) > Discovered (1.0x)
- **🤖 Claude Skill**: AI reads patterns BEFORE writing code
- **📊 Zero-config bootstrap**: Instant pattern detection from existing code
- **🔄 Progressive enhancement**: 95% auto → 4% blessed → 1% golden
- **🌐 Language support**: Go (now), TypeScript/Python (planned)
- **🎨 AST-based matching**: Compares structure, not formatting
- **📈 Continuous learning**: Patterns improve as you merge code
- **🚀 Local + CI/CD**: Works in pre-commit hooks and GitHub Actions
- **⚙️ Per-service config**: Each microservice manages its own patterns
- **🔌 Model agnostic**: Works with any AI coding tool

## License

MIT License - see [LICENSE](LICENSE) for details.
