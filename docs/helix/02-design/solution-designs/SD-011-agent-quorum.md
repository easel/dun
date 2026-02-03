---
dun:
  id: SD-011
  depends_on:
  - F-018
---
# SD-011: Agent Quorum Solution Design

**User Story**: US-011 - Use Agent Quorum for High-Confidence Decisions
**Status**: Planned
**Date**: 2026-01-31

## 1. Overview

This document describes the architecture and implementation plan for the Agent
Quorum feature, which enables running tasks through multiple agent harnesses,
requiring consensus (vote mode), or synthesizing a merged result (synthesis
mode).

### 1.1 Problem Statement

Single-agent execution provides no redundancy or validation. For high-stakes
changes (security patches, data migrations, production deployments), and for
high-quality spec generation, maintainers need higher confidence and richer
outputs than a single agent can provide.

### 1.2 Solution Summary

Extend Dun with:
- `dun loop --quorum` to apply quorum to each iteration prompt.
- `dun quorum` to run a one-shot multi-agent vote and select a response.
- `dun synth` (`dun quorum --synthesize`) to run a one-shot multi-agent draft
  plus synthesis meta-harness.
- Execute tasks through multiple harnesses concurrently or sequentially.
- Compare responses using semantic similarity.
- Apply changes only when quorum is reached (loop mode) or return the chosen
  response (one-shot mode).
- Handle conflicts through escalation, preference, or skip.

## 2. Architecture Overview

### 2.1 Component Diagram

```
+-------------------+     +-------------------+     +-------------------+
| Loop/Quorum Cmds  |---->|  Quorum Manager   |---->|  Result Aggregator|
+-------------------+     +-------------------+     +-------------------+
                                   |                         |
                    +--------------+--------------+          |
                    |              |              |          |
                    v              v              v          v
            +----------+   +----------+   +----------+  +-----------+
            | Harness  |   | Harness  |   | Harness  |  | Semantic  |
            | (persona)|   | (persona)|   | (persona)|  | Comparator|
            +----------+   +----------+   +----------+  +-----------+
                    |              |              |
                    v              v              v
            +-------------------------------------------+
            |           Response Collector              |
            +-------------------------------------------+
                                   |
                                   v
            +-------------------------------------------+
            |           Conflict Resolver               |
            +-------------------------------------------+
                                   |
                                   v
            +-------------------------------------------+
            |        Synthesis Meta-Harness (opt)       |
            +-------------------------------------------+
```

### 2.2 Data Flow

1. **Loop/Quorum Command** parses quorum flags and creates `QuorumConfig`
2. **Quorum Manager** coordinates harness execution based on mode (parallel/sequential)
3. **Harnesses** execute prompts and return `HarnessResult` structs
4. **Response Collector** gathers all results with timing metadata
5. **Semantic Comparator** groups responses by similarity
6. **Result Aggregator** determines if quorum is met
7. **Conflict Resolver** handles disagreements per policy
8. **Synthesis Meta-Harness** (optional) merges drafts into one result

## 3. Quorum Strategies

### 3.1 Strategy Definitions

| Strategy | Type | Behavior | Use Case |
|----------|------|----------|----------|
| `any` | Named | First valid response wins | Speed over accuracy |
| `majority` | Named | >50% must agree | Balanced confidence |
| `unanimous` | Named | All must agree | Maximum confidence |
| `N` | Numeric | At least N must agree | Custom threshold |

### 3.2 Strategy Resolution

```go
func (q *QuorumConfig) IsMet(agreements int, total int) bool {
    switch q.Strategy {
    case "any":
        return agreements >= 1
    case "majority":
        return agreements > total/2
    case "unanimous":
        return agreements == total
    default:
        // Numeric threshold
        return agreements >= q.Threshold
    }
}
```

### 3.3 Validation Rules

- `quorum N` where N > number of harnesses: error
- `quorum 0` or negative: error
- `quorum unanimous` with single harness: warning (degrades to single-agent)
- `quorum majority` with 2 harnesses: requires both (2/2 > 50%)

### 3.4 Quorum Modes

| Mode | Behavior | Surface |
|------|----------|---------|
| `vote` (default) | Select an existing response that meets quorum | `dun quorum`, `dun loop --quorum` |
| `synthesize` | Merge drafts via a synthesis meta-harness | `dun synth` / `dun quorum --synthesize` |

## 4. Multi-Harness Execution

### 4.1 Parallel Mode (Default)

```go
func (qm *QuorumManager) executeParallel(ctx context.Context, prompt string) []HarnessResult {
    var wg sync.WaitGroup
    results := make(chan HarnessResult, len(qm.harnesses))

    for _, h := range qm.harnesses {
        wg.Add(1)
        go func(harness Harness) {
            defer wg.Done()
            start := time.Now()
            resp, err := harness.Execute(ctx, prompt)
            results <- HarnessResult{
                Harness:  harness.Name(),
                Response: resp,
                Error:    err,
                Duration: time.Since(start),
            }
        }(h)
    }

    wg.Wait()
    close(results)
    return collectResults(results)
}
```

### 4.2 Sequential Mode (Cost Optimization)

```go
func (qm *QuorumManager) executeSequential(ctx context.Context, prompt string) []HarnessResult {
    var results []HarnessResult
    agreementGroups := make(map[string][]HarnessResult)

    for _, h := range qm.harnesses {
        start := time.Now()
        resp, err := h.Execute(ctx, prompt)
        result := HarnessResult{
            Harness:  h.Name(),
            Response: resp,
            Error:    err,
            Duration: time.Since(start),
        }
        results = append(results, result)

        if err == nil {
            groupKey := qm.comparator.Normalize(resp)
            agreementGroups[groupKey] = append(agreementGroups[groupKey], result)

            // Early exit if quorum already met
            for _, group := range agreementGroups {
                if qm.config.IsMet(len(group), len(qm.harnesses)) {
                    return results
                }
            }
        }

        // Early exit for unanimous: any disagreement fails
        if qm.config.Strategy == "unanimous" && len(agreementGroups) > 1 {
            return results
        }
    }

    return results
}
```

### 4.3 Harness Interface

```go
type Harness interface {
    Name() string
    Execute(ctx context.Context, prompt string) (string, error)
    SupportsAutomation(mode string) bool
}

type HarnessRegistry struct {
    harnesses map[string]HarnessFactory
}

type HarnessFactory func(config HarnessConfig) Harness
```

## 5. Semantic Comparison Algorithm

### 5.1 Multi-Level Comparison

The comparison system uses a tiered approach:

```go
type SemanticComparator struct {
    normalizer   ResponseNormalizer
    diffEngine   DiffEngine
    threshold    float64  // Default: 0.95
}

func (sc *SemanticComparator) Compare(a, b string) ComparisonResult {
    // Level 1: Exact match after normalization
    normA, normB := sc.normalizer.Normalize(a), sc.normalizer.Normalize(b)
    if normA == normB {
        return ComparisonResult{Match: true, Confidence: 1.0, Level: "exact"}
    }

    // Level 2: Structural similarity (for code/JSON)
    if structural := sc.structuralCompare(a, b); structural.Score >= sc.threshold {
        return ComparisonResult{Match: true, Confidence: structural.Score, Level: "structural"}
    }

    // Level 3: Semantic similarity (for prose)
    if semantic := sc.semanticCompare(a, b); semantic.Score >= sc.threshold {
        return ComparisonResult{Match: true, Confidence: semantic.Score, Level: "semantic"}
    }

    return ComparisonResult{Match: false, Diff: sc.diffEngine.Diff(a, b)}
}
```

### 5.2 Normalization Pipeline

```go
type ResponseNormalizer struct {
    stripWhitespace    bool
    normalizeLineEnds  bool
    sortJSONKeys       bool
    ignoreComments     bool
}

func (rn *ResponseNormalizer) Normalize(s string) string {
    result := s

    // Normalize line endings
    if rn.normalizeLineEnds {
        result = strings.ReplaceAll(result, "\r\n", "\n")
    }

    // Collapse whitespace
    if rn.stripWhitespace {
        result = collapseWhitespace(result)
    }

    // Sort JSON keys for deterministic comparison
    if rn.sortJSONKeys && looksLikeJSON(result) {
        result = sortJSONKeys(result)
    }

    // Strip comments (language-aware)
    if rn.ignoreComments {
        result = stripComments(result)
    }

    return result
}
```

### 5.3 Structural Comparison (Code-Aware)

```go
func (sc *SemanticComparator) structuralCompare(a, b string) SimilarityScore {
    // Parse into lines, ignoring empty lines
    linesA := significantLines(a)
    linesB := significantLines(b)

    // Compute Levenshtein distance on line level
    distance := lineLevenshtein(linesA, linesB)
    maxLen := max(len(linesA), len(linesB))

    if maxLen == 0 {
        return SimilarityScore{Score: 1.0}
    }

    score := 1.0 - float64(distance)/float64(maxLen)
    return SimilarityScore{Score: score}
}
```

### 5.4 Response Grouping

```go
func (ra *ResultAggregator) GroupByAgreement(results []HarnessResult) []ResponseGroup {
    groups := make(map[string]*ResponseGroup)

    for _, r := range results {
        if r.Error != nil {
            continue
        }

        var assigned bool
        for key, group := range groups {
            comparison := ra.comparator.Compare(r.Response, group.Canonical)
            if comparison.Match {
                group.Members = append(group.Members, r)
                group.Confidence = min(group.Confidence, comparison.Confidence)
                assigned = true
                break
            }
        }

        if !assigned {
            key := ra.comparator.Normalize(r.Response)
            groups[key] = &ResponseGroup{
                Canonical:  r.Response,
                Members:    []HarnessResult{r},
                Confidence: 1.0,
            }
        }
    }

    return sortGroupsBySize(groups)
}
```

## 6. Conflict Resolution Logic

### 6.1 Resolution Strategies

```go
type ConflictResolver struct {
    escalate   bool
    prefer     string
    stdin      io.Reader
    stdout     io.Writer
}

func (cr *ConflictResolver) Resolve(groups []ResponseGroup, config QuorumConfig) Resolution {
    largestGroup := groups[0]

    // Check if quorum is met
    if config.IsMet(len(largestGroup.Members), config.TotalHarnesses) {
        return Resolution{
            Outcome:  "accepted",
            Response: largestGroup.Canonical,
            Reason:   fmt.Sprintf("quorum met: %d/%d agree", len(largestGroup.Members), config.TotalHarnesses),
        }
    }

    // Quorum not met - conflict detected
    conflict := cr.buildConflictReport(groups)

    // Strategy 1: Escalate to human
    if cr.escalate {
        return cr.humanReview(conflict)
    }

    // Strategy 2: Prefer specific harness
    if cr.prefer != "" {
        return cr.preferredHarness(groups, cr.prefer)
    }

    // Strategy 3: Default - skip task
    return Resolution{
        Outcome:  "skipped",
        Conflict: conflict,
        Reason:   "quorum not met, no escalation configured",
    }
}
```

### 6.2 Human Escalation

```go
func (cr *ConflictResolver) humanReview(conflict ConflictReport) Resolution {
    fmt.Fprintf(cr.stdout, "\n=== QUORUM CONFLICT ===\n")
    fmt.Fprintf(cr.stdout, "Task: %s\n\n", conflict.TaskID)

    for i, group := range conflict.Groups {
        fmt.Fprintf(cr.stdout, "Option %d (%d harnesses: %s):\n",
            i+1, len(group.Members), harnessNames(group.Members))
        fmt.Fprintf(cr.stdout, "```\n%s\n```\n\n", truncate(group.Canonical, 500))
    }

    fmt.Fprintf(cr.stdout, "Enter choice (1-%d), 's' to skip, 'q' to quit: ", len(conflict.Groups))

    var choice string
    fmt.Fscanln(cr.stdin, &choice)

    switch choice {
    case "s":
        return Resolution{Outcome: "skipped", Reason: "user skipped"}
    case "q":
        return Resolution{Outcome: "aborted", Reason: "user quit"}
    default:
        idx, err := strconv.Atoi(choice)
        if err != nil || idx < 1 || idx > len(conflict.Groups) {
            return Resolution{Outcome: "skipped", Reason: "invalid choice"}
        }
        return Resolution{
            Outcome:  "accepted",
            Response: conflict.Groups[idx-1].Canonical,
            Reason:   "user selected option " + choice,
        }
    }
}
```

### 6.3 Conflict Report Structure

```go
type ConflictReport struct {
    TaskID     string           `json:"task_id"`
    Timestamp  time.Time        `json:"timestamp"`
    Groups     []ResponseGroup  `json:"groups"`
    Diffs      []GroupDiff      `json:"diffs"`
    Harnesses  []string         `json:"harnesses"`
    Quorum     QuorumConfig     `json:"quorum"`
}

type GroupDiff struct {
    GroupA  int    `json:"group_a"`
    GroupB  int    `json:"group_b"`
    Unified string `json:"unified_diff"`
}
```

## 7. Data Structures

### 7.1 Core Types

```go
// QuorumConfig holds quorum settings parsed from flags
type QuorumConfig struct {
    Strategy       string        // "any", "majority", "unanimous", or ""
    Threshold      int           // Numeric threshold when Strategy is ""
    Harnesses      []HarnessSpec // Harness + persona specs
    TotalHarnesses int           // Computed count
    Mode           string        // "parallel" or "sequential"
    Prefer         string        // Preferred harness on conflict
    Escalate       bool          // Pause for human review on conflict
    Similarity     float64       // Similarity threshold for grouping
    Synthesize     bool          // Enable synthesis mode
    Synthesizer    SynthSpec      // Meta-harness config for synthesis
}

// HarnessSpec captures a harness with an optional persona and model override.
type HarnessSpec struct {
    Name     string `json:"name"`
    Persona  string `json:"persona,omitempty"`
    Model    string `json:"model,omitempty"`
}

// SynthSpec defines the synthesis meta-harness configuration.
type SynthSpec struct {
    Name    string `json:"name"`
    Persona string `json:"persona,omitempty"`
    Model   string `json:"model,omitempty"`
    Prompt  string `json:"prompt,omitempty"`
}

// HarnessResult captures a single harness execution
type HarnessResult struct {
    Harness   string        `json:"harness"`
    Response  string        `json:"response"`
    Error     error         `json:"error,omitempty"`
    Duration  time.Duration `json:"duration_ms"`
    Timestamp time.Time     `json:"timestamp"`
}

// ResponseGroup represents harnesses that agree
type ResponseGroup struct {
    Canonical  string          `json:"canonical"`
    Members    []HarnessResult `json:"members"`
    Confidence float64         `json:"confidence"`
}

// Resolution is the outcome of quorum evaluation
type Resolution struct {
    Outcome  string         `json:"outcome"` // "accepted", "skipped", "aborted"
    Response string         `json:"response,omitempty"`
    Conflict *ConflictReport `json:"conflict,omitempty"`
    Reason   string         `json:"reason"`
}

// AgreementStats tracks long-term agreement rates
type AgreementStats struct {
    HarnessID       string    `json:"harness_id"`
    TotalTasks      int       `json:"total_tasks"`
    AgreementCount  int       `json:"agreement_count"`
    DisagreementCount int     `json:"disagreement_count"`
    LastUpdated     time.Time `json:"last_updated"`
}
```

### 7.2 Configuration Extension

```go
// Extend existing Config struct
type Config struct {
    Version string       `yaml:"version"`
    Agent   AgentConfig  `yaml:"agent"`
    Quorum  QuorumYAML   `yaml:"quorum"` // NEW
}

type QuorumYAML struct {
    Default     string        `yaml:"default"`     // Default strategy
    Mode        string        `yaml:"mode"`        // "parallel" or "sequential"
    Similarity  float64       `yaml:"similarity"`  // Similarity threshold
    Prefer      string        `yaml:"prefer"`      // Default preferred harness
    Escalate    bool          `yaml:"escalate"`    // Default escalation behavior
    Harnesses   []HarnessSpec `yaml:"harnesses"`   // Default harnesses + personas
    Synthesize  bool          `yaml:"synthesize"`  // Default to synthesis mode
    Synthesizer SynthSpec     `yaml:"synthesizer"` // Meta-harness config
}
```

**Persona Registry Boundary**: persona definitions (system prompts and defaults)
live in the harness/DDX layer. Dun only references persona names and passes
them to harnesses.
Quorum vote and synthesis prompts are owned by Dun (not agents) and are passed
as task prompts to the harnesses.

## 8. File Structure

### 8.1 New Files

| File | Purpose |
|------|---------|
| `cmd/dun/quorum.go` | New `dun quorum` / `dun synth` command surface |
| `internal/dun/quorum.go` | QuorumManager, QuorumConfig, strategy logic |
| `internal/dun/quorum_test.go` | Unit tests for quorum logic |
| `internal/dun/harness.go` | Harness interface, registry, implementations |
| `internal/dun/harness_test.go` | Harness unit tests |
| `internal/dun/semantic.go` | SemanticComparator, normalizer, diff |
| `internal/dun/semantic_test.go` | Semantic comparison tests |
| `internal/dun/conflict.go` | ConflictResolver, ConflictReport |
| `internal/dun/conflict_test.go` | Conflict resolution tests |
| `internal/dun/stats.go` | AgreementStats tracking |
| `internal/dun/stats_test.go` | Stats tests |
| `internal/testdata/repos/quorum-basic/` | Test fixture: simple project |
| `internal/testdata/repos/quorum-conflict/` | Test fixture: conflict scenarios |

### 8.2 Modified Files

| File | Changes |
|------|---------|
| `cmd/dun/main.go` | Add quorum flags to loop command, integrate QuorumManager |
| `internal/dun/types.go` | Add QuorumConfig, HarnessResult, Resolution types |
| `internal/dun/config.go` | Add QuorumYAML to Config, update ApplyConfig |
| `internal/dun/exitcodes.go` | Add ExitQuorumConflict (5), ExitQuorumAborted (6) |

## 9. Implementation Phases

### Phase 1: Foundation (3 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P1.1 | Define core types | `types.go` | 1h |
| P1.2 | Implement QuorumConfig parsing | `quorum.go` | 2h |
| P1.3 | Add flag parsing to loop command | `main.go` | 2h |

**Deliverable**: `dun loop --quorum 2 --harnesses claude,gemini` parses without error.

### Phase 2: Harness Abstraction (4 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P2.1 | Define Harness interface | `harness.go` | 1h |
| P2.2 | Refactor existing harness code to interface | `harness.go`, `main.go` | 3h |
| P2.3 | Implement HarnessRegistry | `harness.go` | 2h |
| P2.4 | Create mock harness for testing | `harness_test.go` | 2h |

**Deliverable**: Harness calls go through interface; mocks work in tests.

### Phase 3: Parallel Execution (3 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P3.1 | Implement parallel execution | `quorum.go` | 3h |
| P3.2 | Add timeout and cancellation | `quorum.go` | 2h |
| P3.3 | Collect results with timing metadata | `quorum.go` | 1h |

**Deliverable**: Multiple harnesses execute concurrently; results collected.

### Phase 4: Semantic Comparison (4 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P4.1 | Implement ResponseNormalizer | `semantic.go` | 2h |
| P4.2 | Implement exact and structural comparison | `semantic.go` | 3h |
| P4.3 | Implement response grouping | `semantic.go` | 2h |
| P4.4 | Add configurable threshold | `semantic.go`, `config.go` | 1h |

**Deliverable**: Responses are grouped by semantic similarity.

### Phase 5: Quorum Evaluation (3 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P5.1 | Implement strategy evaluation | `quorum.go` | 2h |
| P5.2 | Integrate with loop command | `main.go` | 3h |
| P5.3 | Add quorum result logging | `quorum.go` | 1h |

**Deliverable**: Quorum is evaluated; changes applied only when met.

### Phase 6: Conflict Resolution (4 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P6.1 | Implement ConflictReport generation | `conflict.go` | 2h |
| P6.2 | Implement escalation (human review) | `conflict.go` | 3h |
| P6.3 | Implement prefer strategy | `conflict.go` | 1h |
| P6.4 | Implement default skip behavior | `conflict.go` | 1h |

**Deliverable**: Conflicts are handled per configuration.

### Phase 7: Sequential/Cost Mode (2 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P7.1 | Implement sequential execution | `quorum.go` | 3h |
| P7.2 | Add early exit optimization | `quorum.go` | 2h |

**Deliverable**: `--cost-optimized` runs harnesses sequentially with early exit.

### Phase 8: Statistics Tracking (2 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P8.1 | Implement AgreementStats storage | `stats.go` | 2h |
| P8.2 | Add stats query command | `main.go` | 2h |

**Deliverable**: Agreement rates tracked and queryable.

### Phase 9: Testing and Polish (3 tasks)

| Task | Description | Files | Estimate |
|------|-------------|-------|----------|
| P9.1 | Create test fixtures | `testdata/` | 2h |
| P9.2 | Integration tests | `*_test.go` | 4h |
| P9.3 | Documentation and help text | `main.go` | 1h |

**Deliverable**: Full test coverage; documentation complete.

## 10. CLI Interface

### 10.1 New Flags

```
dun loop [existing flags] [quorum flags]

Quorum Flags:
  --quorum <strategy|N>   Quorum strategy: any, majority, unanimous, or number
                          (default: none - single harness mode)
  --harnesses <list>      Comma-separated harness names (supports name@persona)
  --cost-optimized             Run harnesses sequentially, stop on quorum
  --escalate              Pause for human review on conflict
  --prefer <harness>      Use this harness response on conflict
  --similarity <float>    Semantic similarity threshold (default: 0.95)
```

### 10.2 Example Commands

```bash
# Basic quorum: 2 of 3 must agree
dun loop --harnesses claude,gemini,codex --quorum 2

# Unanimous agreement required
dun loop --harnesses claude,gemini --quorum unanimous

# Cost-optimized: stop when 2 agree
dun loop --harnesses claude,gemini,codex --quorum 2 --cost-optimized

# With escalation on conflict
dun loop --harnesses claude,gemini --quorum unanimous --escalate

# With preferred fallback
dun loop --harnesses claude,gemini,codex --quorum majority --prefer claude

# One-shot quorum
dun quorum --task \"Write the quorum spec\" --harnesses codex@architect,claude@critic --quorum majority

# One-shot synthesis
dun synth --task \"Write the quorum spec\" --harnesses codex@architect,claude@critic --synthesizer codex@editor
```

### 10.3 Exit Codes

| Code | Constant | Meaning |
|------|----------|---------|
| 0 | ExitSuccess | All checks pass or quorum met |
| 1 | ExitCheckFailed | Check failed |
| 5 | ExitQuorumConflict | Quorum not met, task skipped |
| 6 | ExitQuorumAborted | User aborted during escalation |

## 11. Performance Considerations

### 11.1 Parallel Execution

- **Goroutine pool**: Limit concurrent harness calls to prevent resource exhaustion
- **Context cancellation**: Cancel remaining harnesses when quorum is met (optional optimization)
- **Timeout handling**: Per-harness timeout with overall deadline

### 11.2 Memory Management

- **Response streaming**: Consider streaming for large responses
- **Diff computation**: Lazy computation of diffs (only when needed for conflict report)
- **Stats persistence**: Batch writes to reduce I/O

### 11.3 Network Efficiency

- **Connection reuse**: Keep HTTP connections alive between iterations
- **Retry with backoff**: Handle transient failures without cascade
- **Rate limiting**: Respect per-harness API limits

### 11.4 Cost Optimization

- **Sequential mode**: Default for expensive harnesses
- **Caching**: Cache identical prompts within session (optional)
- **Token estimation**: Estimate cost before execution (future)

## 12. Risk Assessment

### 12.1 Technical Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Semantic comparison too strict | Rejects valid agreements | Medium | Tunable threshold, multiple comparison levels |
| Semantic comparison too loose | Accepts false agreements | Medium | Conservative default (0.95), manual review option |
| Harness timeouts causing delays | Loop stalls | Medium | Aggressive timeouts, context cancellation |
| API rate limits | Execution failures | High | Exponential backoff, sequential mode |
| Memory pressure from large responses | OOM | Low | Response size limits, streaming |

### 12.2 Operational Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| 3x cost increase with quorum | Unexpected bills | High | Clear documentation, cost-optimized default |
| User confusion on conflict resolution | Stuck loops | Medium | Good defaults, clear prompts |
| Stats storage grows unbounded | Disk usage | Low | Retention policy, compaction |

### 12.3 Scope Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Semantic comparison scope creep | Delayed delivery | Medium | Start with exact/structural only |
| Harness-specific quirks | Inconsistent behavior | Medium | Clear interface contract, mocks |

## 13. Testing Strategy

### 13.1 Unit Tests

- QuorumConfig parsing (all strategies, edge cases)
- Strategy evaluation (IsMet for all combinations)
- ResponseNormalizer (whitespace, JSON, comments)
- SemanticComparator (exact, structural, threshold boundary)
- ConflictResolver (escalate, prefer, skip)

### 13.2 Integration Tests

- Full loop with mock harnesses returning identical responses
- Full loop with mock harnesses returning different responses
- Sequential mode with early exit
- Escalation flow with simulated user input
- Config file loading with quorum settings

### 13.3 Test Fixtures

- `quorum-basic/`: Simple Go project for happy-path testing
- `quorum-conflict/`: Project that triggers predictable harness disagreement

## 14. Open Questions

1. **Stats persistence format**: SQLite, JSON file, or memory-only?
2. **Semantic comparison depth**: Should we support LLM-based similarity for prose?
3. **Harness ordering in sequential mode**: Alphabetical, by cost, or configurable?
4. **Conflict report format**: Plain text, JSON, or both?
5. **Quorum per-check or per-iteration**: Should different checks have different quorum requirements?

## 15. Dependencies

- No new external dependencies required
- Uses existing harness execution code (refactored to interface)
- Standard library: `sync`, `context`, `time`, `strings`

## 16. References

- [US-011: Agent Quorum User Story](../../01-frame/user-stories/US-011-agent-quorum.md)
- [TP-011: Agent Quorum Test Plan](../../03-test/test-plans/TP-011-agent-quorum.md)
- Existing implementation: `cmd/dun/main.go` (runLoop, callHarness)
