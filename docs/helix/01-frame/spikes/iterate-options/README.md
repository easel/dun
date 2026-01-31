# Iterate Options Spike

Comparing different approaches for closing the autonomous loop.

## Key Metric: Context Available for Real Work

See `context-overhead.md` for detailed analysis. The key insight:

```
Effective Work Capacity = Context Window - Framework Overhead
```

| Approach | Overhead | 200K Effective | Notes |
|----------|----------|----------------|-------|
| Direct API | ~100 tokens | 99.95% | Minimal, just your prompt |
| CLI (claude/gemini/codex) | ~500 tokens | 99.75% | Small system prompt |
| Claude Code session | ~10-20K tokens | 90-95% | CLAUDE.md + tools + history |
| Task tool (nested) | ~5-10K tokens | 95-97% | Agent template + parent context |

**Winner for autonomous loops: CLI or direct API with fresh context each iteration.**

## Test Prompt

Simple task to test mechanics:
```
Pick your favorite word, make it bold, move to OUTPUT.
INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:
```

## Options

### Claude (Anthropic)

| Option | Driver | Executor | Context | Complexity |
|--------|--------|----------|---------|------------|
| A | Bash loop | Claude CLI | Fresh each iteration | Lowest |
| B | Go binary | Claude CLI subprocess | Fresh each iteration | Low |
| C | Claude Code | Task tool (agents) | Shared session | Medium |
| D | Go binary | Anthropic API direct | Managed by Go | Highest |

### Multi-Harness

| Option | Provider | Method | Context Window |
|--------|----------|--------|----------------|
| E | Google | Gemini CLI/API | 1M-2M |
| F | OpenAI | Codex/GPT CLI | 128K-200K |

## Files

- `option-a.sh` - Pure bash loop (Ralph pattern)
- `option-b.go` - Go shelling out to Claude CLI
- `option-c-prompt.md` - Prompt for Claude Code + Task tool
- `option-d.go` - Direct API calls (sketch only)
- `option-e-gemini.sh` - Gemini CLI/Python
- `option-f-codex.sh` - OpenAI/Codex CLI/Python
- `measure-context.go` - Context overhead measurement tool
- `context-overhead.md` - Analysis of context efficiency
- `test-input.txt` - Test data

## Running

```bash
# Option A (Claude CLI bash loop)
./option-a.sh

# Option B (Go + Claude CLI)
go run option-b.go

# Option C (Claude Code Task tool)
# Run from Claude Code session with option-c-prompt.md

# Option D (Direct Anthropic API)
ANTHROPIC_API_KEY=sk-... go run option-d.go

# Option E (Gemini)
GOOGLE_API_KEY=... ./option-e-gemini.sh

# Option F (OpenAI/Codex)
OPENAI_API_KEY=sk-... ./option-f-codex.sh

# Measure context overhead
ANTHROPIC_API_KEY=sk-... go run measure-context.go --provider anthropic
OPENAI_API_KEY=sk-... go run measure-context.go --provider openai
```

## Test Results

### Task Tool (Option C) - Tested

```
INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT: **grape**
```

Works. Context shared with parent session.

### Next: Measure actual token overhead for each approach.
