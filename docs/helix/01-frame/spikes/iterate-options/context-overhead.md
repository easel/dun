# Context Overhead Analysis

The real question: **How much context is available for actual work?**

## Why This Matters

Each approach consumes context window before your task even starts:

| Approach | Overhead Source | Estimated Tokens |
|----------|-----------------|------------------|
| Direct API | Just your prompt | ~50 |
| Claude CLI | Minimal system prompt | ~500 |
| Claude Code | CLAUDE.md + tools + history | ~5,000-20,000 |
| Task tool (from CC) | Agent template + parent context | ~2,000-10,000 |

With a 200K context window, overhead matters less. But for **fresh context per iteration** (Ralph pattern), minimizing overhead maximizes work capacity.

## Measurement Approach

### 1. Token Counting

```bash
# Count tokens in system prompt / CLAUDE.md
# Using tiktoken or Anthropic's tokenizer

# For CLAUDE.md
wc -w CLAUDE.md  # rough: words * 1.3 ≈ tokens

# For API calls, check usage.input_tokens in response
```

### 2. Effective Context Ratio

```
Effective Ratio = (Context Window - Overhead) / Context Window

200K window, 20K overhead = 90% effective
200K window, 5K overhead = 97.5% effective
200K window, 500 overhead = 99.75% effective
```

### 3. Practical Test

Send the same task with increasing input sizes until failure:

```
Test 1: 100 words → measure tokens used
Test 2: 1,000 words → measure tokens used
Test 3: 10,000 words → measure tokens used
...
Test N: Find max input size before context limit
```

## Multi-Harness Comparison

### Claude (Anthropic)

| Method | Context | Overhead | Effective |
|--------|---------|----------|-----------|
| Direct API | 200K | ~100 | 99.95% |
| Claude CLI | 200K | ~500 | 99.75% |
| Claude Code | 200K | ~10-20K | 90-95% |
| Task tool | 200K | ~5-10K | 95-97% |

### Gemini (Google)

| Method | Context | Notes |
|--------|---------|-------|
| Gemini API | 1M-2M | Largest context window |
| AI Studio | 1M | Browser-based |
| Gemini CLI | 1M | `gemini` command |

### Codex/GPT (OpenAI)

| Method | Context | Notes |
|--------|---------|-------|
| o3 API | 128K-200K | Reasoning model |
| GPT-4o API | 128K | Fast, multimodal |
| Codex CLI | varies | Depends on backend model |

## Key Insight

**Fresh context = maximum effective capacity**

The Ralph pattern's genius is resetting to zero overhead each iteration. Context accumulation is the enemy:

```
Iteration 1: 5K overhead + 10K work = 15K total
Iteration 2: 15K overhead + 10K work = 25K total
Iteration 3: 25K overhead + 10K work = 35K total
...
Iteration 20: 195K overhead + 10K work = CONTEXT LIMIT
```

With fresh context:
```
Every iteration: 5K overhead + 10K work = 15K total (reset)
```

## Recommendation

For autonomous loops, prioritize:

1. **Fresh context each iteration** (Ralph pattern)
2. **Minimal overhead** (direct API or CLI over Claude Code)
3. **Large context models** (Gemini 1M+ if overhead is unavoidable)

The ideal `dun iterate` approach:
- Generate minimal prompt (just task + instructions)
- Call via CLI or API (not nested in Claude Code session)
- Parse structured output
- Repeat with fresh context
