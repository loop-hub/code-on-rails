# Code on Rails

**AI-first code consistency with a complete feedback loop.**

Code on Rails (`cr`) learns your codebase patterns and ensures AI-generated code matches your team's standards. Works with Claude, Copilot, Cursor, and any AI assistant.

## 🔄 The Feedback Loop (Key Feature)

The killer feature of Code on Rails is the **fully automatic feedback loop** - zero manual steps required:

```
┌─────────────────────────────────────────────────────────────────┐
│                  AUTOMATIC AI FEEDBACK LOOP                     │
│                                                                 │
│   ┌──────────┐              ┌──────────┐                       │
│   │  Claude  │──── push ───▸│  GitHub  │                       │
│   │  writes  │              │    CI    │                       │
│   │   code   │              └────┬─────┘                       │
│   └──────────┘                   │                             │
│        ▲                         │ commits                     │
│        │                         ▼                             │
│        │              ┌─────────────────────┐                  │
│        └──── reads ───│ .code-on-rails-     │                  │
│                       │  feedback.json      │                  │
│                       └─────────────────────┘                  │
│                                                                 │
│   Claude automatically reads feedback file and fixes issues    │
└─────────────────────────────────────────────────────────────────┘
```

### How It Works (Zero Manual Steps)

1. **Claude writes code** and pushes to branch
2. **GitHub CI runs** `cr check` and finds pattern deviations
3. **CI commits** `.code-on-rails-feedback.json` to the branch
4. **Claude reads the feedback file** automatically (via skill)
5. **Claude fixes the issues** and pushes again
6. **CI passes** - feedback file removed

### Two Ways to Share Patterns

**1. Team with Shared Skills (Recommended)**
```bash
# Generate portable skills file for your team
cr learn --update-skills

# Commit .code-on-rails-skills.json to repo
# Everyone's AI generates code matching team patterns
```

**2. Enterprise with Central Skills Repository**
```bash
# Store skills in a central repo
cr learn --update-skills -s skills/go-microservices.json

# Teams import skills from the central repo
# Consistent patterns across all services
```

## 🚀 GitHub Workflow Integration (Key Feature)

Code on Rails provides out-of-the-box GitHub Actions integration with AI feedback artifacts:

### Quick Setup

```yaml
# .github/workflows/ai-feedback-loop.yml
name: AI Feedback Loop

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: read
  pull-requests: write

jobs:
  analyze-and-feedback:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build and Run Code on Rails
        run: |
          go build -o cr ./cmd/cr
          ./cr check --format github > comment.txt
          ./cr feedback -o cr-ai-feedback.json

      - name: Upload AI Feedback Artifact
        uses: actions/upload-artifact@v4
        with:
          name: ai-feedback
          path: cr-ai-feedback.json

      - name: Post PR Comment
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const comment = fs.readFileSync('comment.txt', 'utf8');
            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              body: comment
            });
```

### What You Get

**PR Comment:**
```
## 🤖 Code on Rails - AI Code Review

**5 files** analyzed | **4** auto-approved | **1** needs review

<details>
<summary>✅ Auto-approved (4 files, 234 lines)</summary>
| File | Pattern | Match |
|------|---------|-------|
| `src/handlers/user.go` | http_handler | 98% |
...
</details>

### 🔍 Needs Human Review

#### `src/services/payment.go`
**Pattern:** `service` (76% match)
**Issues found:**
- ⚠️ **error_handling** at line 45
  - Expected: `errors.Wrap`
  - 💡 Wrap errors with context using errors.Wrap()

---
### 🔄 AI Feedback Loop

📝 **Feedback file committed to branch**: `.code-on-rails-feedback.json`

Claude will automatically read this file and fix the issues.
Just ask: *"Check for Code on Rails feedback and fix any issues"*
```

**AI Feedback Artifact (cr-ai-feedback.json):**
```json
{
  "summary": {
    "total_files": 5,
    "needs_fixes": 1,
    "auto_approved": 4,
    "primary_language": "go"
  },
  "files_to_fix": [
    {
      "file_path": "src/services/payment.go",
      "pattern_type": "service",
      "match_score": 76,
      "issues": [
        {
          "type": "different",
          "element": "error_handling",
          "expected": "errors.Wrap",
          "suggestion": "Wrap errors with context"
        }
      ],
      "reference_file": "src/services/user_service.go"
    }
  ],
  "instructions": "INSTRUCTIONS FOR FIXING CODE:\n1. Read reference_file..."
}
```

## Quick Start

```bash
# Install
go install github.com/loop-hub/code-on-rails/cmd/cr@latest

# Initialize (discovers patterns automatically)
cd your-project/
cr init

# Check AI-generated code
cr check

# Get AI-readable feedback for fixes
cr feedback

# Generate skills file for team sharing
cr learn --update-skills
```

## Commands

| Command | Description |
|---------|-------------|
| `cr init` | Bootstrap patterns from existing codebase |
| `cr check` | Validate code against established patterns |
| `cr check --format github` | Output rich markdown for PR comments |
| `cr check --format json` | Output JSON for programmatic access |
| `cr feedback` | Generate AI-readable feedback for fixing issues |
| `cr feedback -o file.json` | Save feedback to file |
| `cr learn` | Update patterns from merged code |
| `cr learn --update-skills` | Generate portable skills file |
| `cr bless <file>` | Mark a file as a blessed pattern example |

## How It Works

### 1. Pattern Discovery

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

### 2. Pattern Validation

```bash
$ cr check

Analyzing AI-generated code...

✓ internal/handlers/product_handler.go
  Pattern: HTTP Handler (98% match)
  Auto-approved

⚠ internal/services/product_service.go
  Pattern: Service Layer (87% match)
  Deviations:
    - Different error handling (expected errors.Wrap, found fmt.Errorf)
  Needs review
```

### 3. AI Feedback Generation

```bash
$ cr feedback

{
  "summary": {
    "total_files": 2,
    "needs_fixes": 1,
    "auto_approved": 1
  },
  "files_to_fix": [...],
  "pattern_examples": [...],
  "instructions": "1. Read reference_file to understand pattern..."
}
```

## Configuration

Each project has a `.code-on-rails.yml` file:

```yaml
version: "1.0"
language: go
ai_source: any

patterns:
  - id: http_handler_pattern
    name: HTTP Handler
    type: http_handler

    # 🏆 Golden examples (highest priority)
    annotated_golden:
      - path: internal/handlers/user_handler.go
        blessed_by: "@senior-dev"
        reason: "Template for all handlers"

    # ⭐ Config-blessed (high priority)
    config_blessed:
      - path: internal/handlers/auth_handler.go
        blessed_by: "config"

    # 📊 Auto-discovered (normal priority)
    discovered:
      - path: internal/handlers/product_handler.go
        similarity_score: 0.95

settings:
  auto_approve_threshold: 95
  learn_on_merge: true

detection:
  method: heuristic  # Uses AI code characteristics
```

## Language Support

| Language | Status | Patterns Detected |
|----------|--------|-------------------|
| Go | ✅ Full | handlers, services, repositories, middleware, models |
| TypeScript | ✅ Full | components, hooks, contexts, pages, API routes, stores |
| React | ✅ Full | components, hooks, contexts, styled-components |
| JavaScript | ✅ Basic | Same as TypeScript |

## Features

- **🔄 Complete Feedback Loop**: AI writes → check → AI-readable feedback → AI fixes
- **📦 GitHub Workflow Ready**: Out-of-the-box PR comments and feedback artifacts
- **🎯 Skills Sharing**: Generate portable skills files for team/enterprise use
- **🤖 AI-Optimized Output**: Structured JSON feedback AI assistants can act on
- **📊 Zero-config bootstrap**: Instant pattern detection from existing code
- **🎨 Multi-language**: Go, TypeScript, React, JavaScript support
- **⚡ Heuristic detection**: Automatically identifies AI-generated code
- **🏆 Tiered examples**: Golden (2x) → Blessed (1.5x) → Discovered (1x)
- **📈 Continuous learning**: Patterns improve as you merge code
- **🚀 Local + CI/CD**: Works in pre-commit hooks and GitHub Actions

## The Complete Workflow

```bash
# 1. Developer asks Claude to write code
"Create a payment service"

# 2. Claude writes code and pushes to branch

# 3. GitHub CI runs automatically:
#    - Analyzes code with cr check
#    - Commits .code-on-rails-feedback.json to branch (if issues found)
#    - Posts PR comment with summary

# 4. Developer asks Claude to fix (or Claude reads feedback automatically)
"Check for Code on Rails feedback and fix any issues"

# 5. Claude reads .code-on-rails-feedback.json
#    - Studies reference files
#    - Fixes each issue
#    - Deletes feedback file
#    - Pushes fixes

# 6. CI passes ✅ - all patterns match
```

**With the Claude skill installed, step 4-5 happens automatically!**

## License

MIT License - see [LICENSE](LICENSE) for details.
