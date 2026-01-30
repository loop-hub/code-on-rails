# Installing the Code on Rails Claude Skill

## What This Does

The Code on Rails skill teaches Claude Code to automatically follow your codebase patterns BEFORE writing code, creating a powerful feedback loop:

```
You write patterns → cr learns them → Claude Code reads them → Claude follows them → Less review needed!
```

## Installation

### Option 1: User Skill (Recommended)

Copy the skill to your Claude user skills directory:

```bash
# Find your Claude skills directory
# Usually: ~/.claude/skills/ or ~/Library/Application Support/Claude/skills/

# Copy the skill
cp -r claude-skill ~/.claude/skills/code-on-rails

# Or create a symlink to keep it updated
ln -s "$(pwd)/claude-skill" ~/.claude/skills/code-on-rails
```

### Option 2: Project-Specific Skill

For a specific project:

```bash
# In your project root
mkdir -p .claude/skills
cp -r /path/to/code-on-rails/claude-skill .claude/skills/code-on-rails

# Or symlink
ln -s /path/to/code-on-rails/claude-skill .claude/skills/code-on-rails
```

### Option 3: Global Installation (Team)

For organization-wide patterns:

```bash
# Host the skill in your org's skill repository
git clone https://github.com/your-org/claude-skills
cp -r claude-skill claude-skills/code-on-rails
cd claude-skills
git commit -m "Add Code on Rails skill"
git push

# Team members can then install it
# Via Claude Code's skill manager
```

## Verification

1. Open Claude Code
2. Navigate to a project with `.code-on-rails.yml`
3. Ask: "Create a new user handler"
4. Claude should now automatically:
   - Read `.code-on-rails.yml`
   - View reference implementations
   - Generate code matching the pattern
   - Self-check with `cr check`

## How It Works

When you ask Claude Code to generate code:

1. **Skill triggers** automatically (sees `.code-on-rails.yml`)
2. **Reads patterns** from your config file
3. **Studies references** (the example files)
4. **Generates code** matching the pattern
5. **Self-validates** using `cr check`

## Example Session

**Without the skill:**
```
You: Create a product handler
Claude: [generates generic handler]
cr check: 67% match, missing validation, different error handling
Result: Needs rework
```

**With the skill:**
```
You: Create a product handler
Claude: [reads .code-on-rails.yml]
        [views internal/handlers/user_handler.go]
        [generates handler matching that pattern]
cr check: 98% match
Result: Auto-approved!
```

## Configuration

The skill automatically reads `.code-on-rails.yml` in your project. No additional configuration needed!

But you can customize behavior in your config:

```yaml
# .code-on-rails.yml

settings:
  auto_approve_threshold: 95.0  # How strict
  require_self_check: true      # Claude runs cr check before finishing
  
patterns:
  - id: http_handler_pattern
    examples:
      - "internal/handlers/user_handler.go"  # Claude will study this
    structure:
      required:
        - "net/http"
        - "encoding/json"
```

## Team Workflow

1. **Initialize patterns** in each microservice:
   ```bash
   cr init
   ```

2. **Install skill** (once per developer):
   ```bash
   cp -r claude-skill ~/.claude/skills/code-on-rails
   ```

3. **Use Claude Code** normally:
   ```
   "Add user authentication endpoints"
   "Create product search service"
   "Implement order repository"
   ```

4. **Claude follows patterns automatically** - No manual reminders needed!

5. **Review is faster** - Code already matches patterns

## Benefits

### Before (without skill):
- Claude generates generic code
- Doesn't match your patterns
- Low auto-approval rate (60-70%)
- Lots of review cycles

### After (with skill):
- Claude reads your patterns first
- Generates code matching them
- High auto-approval rate (90-95%)
- Minimal review needed

## Troubleshooting

### "Skill not loading"

Check skill location:
```bash
ls ~/.claude/skills/code-on-rails/SKILL.md
```

Verify it's readable:
```bash
cat ~/.claude/skills/code-on-rails/SKILL.md | head
```

### "Claude not following patterns"

Make sure `.code-on-rails.yml` exists:
```bash
ls .code-on-rails.yml
```

Try explicitly:
```
You: "Create a handler following our patterns"
```

### "Patterns not found"

Initialize first:
```bash
cr init
```

## Advanced: Custom Instructions

You can add project-specific instructions to the skill:

```bash
# Edit the skill
vi ~/.claude/skills/code-on-rails/SKILL.md

# Add custom patterns section:
## Custom Patterns for [Your Company]

### API Response Format
All APIs must return:
{
  "data": {},
  "error": null,
  "meta": {}
}

### Error Handling
Use our custom error package:
github.com/yourco/errors
```

## Integration with CI/CD

The skill and `cr check` work together:

1. **Skill** - Claude follows patterns when writing
2. **cr check** - Validates in CI/CD

```yaml
# .github/workflows/code-on-rails.yml
- name: Check patterns
  run: cr check
  # Should pass because Claude already followed them!
```

## Updating the Skill

When patterns evolve:

```bash
# Update your patterns
vi .code-on-rails.yml

# Or re-learn
cr learn

# Skill automatically reads latest config
# No skill update needed!
```

The skill always reads the current `.code-on-rails.yml`, so it stays in sync automatically.

## FAQ

**Q: Does this slow down Claude?**
A: Minimal. Reading config + 1-2 reference files adds ~5 seconds.

**Q: What if I don't want Claude to follow patterns for this task?**
A: Say "ignore the patterns for this" or temporarily remove `.code-on-rails.yml`

**Q: Can I have different patterns per project?**
A: Yes! Each project has its own `.code-on-rails.yml`

**Q: Does this work with other AI tools?**
A: This skill is Claude Code specific, but the pattern detection works with any AI tool (Copilot, Cursor, etc.)

**Q: What if the pattern is wrong?**
A: Update `.code-on-rails.yml` or the reference file. Claude will follow the updated pattern.

---

## Complete Feedback Loop

```
┌─────────────────────────────────────────────┐
│  1. Your team writes initial code           │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  2. cr init extracts patterns               │
│     → Creates .code-on-rails.yml            │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  3. Claude Skill reads patterns             │
│     → Studies reference files               │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  4. Claude generates code matching patterns │
│     → Auto-validates with cr check          │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  5. cr check in CI validates                │
│     → 95%+ match = auto-approve             │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│  6. cr learn updates patterns               │
│     → Loop continues!                       │
└─────────────────────────────────────────────┘
```

Now you have **both sides** of the equation:
- ✅ **Validation** (cr check)
- ✅ **Prevention** (Claude Skill)

Install the skill and let Claude Code become your team's best developer! 🚂✨
