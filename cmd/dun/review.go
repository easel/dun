package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/easel/dun/internal/dun"
)

type reviewDoc struct {
	Path    string
	Content string
}

type reviewResponse struct {
	Harness  string
	Response string
	Err      error
}

func runReview(args []string, stdout io.Writer, stderr io.Writer) int {
	root := resolveRoot(".")
	explicitConfig := findConfigFlag(args)
	opts := dun.DefaultOptions()
	cfg, loaded, err := dun.LoadConfig(root, explicitConfig)
	if err != nil {
		fmt.Fprintf(stderr, "dun review failed: config error: %v\n", err)
		return dun.ExitConfigError
	}
	if loaded {
		opts = dun.ApplyConfig(opts, cfg)
	}

	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", explicitConfig, "path to config file (default .dun/config.yaml if present; also loads user config)")
	principlesPath := fs.String("principles", "docs/helix/01-frame/principles.md", "path to principles document")
	harnessesFlag := fs.String("harnesses", "", "comma-separated list of review harnesses")
	synthHarness := fs.String("synth-harness", "", "harness used to synthesize final review (default: first harness)")
	model := fs.String("model", opts.AgentModel, "model override for selected harness(es)")
	models := fs.String("models", "", "per-harness model overrides (e.g., codex:o3,claude:sonnet)")
	automation := fs.String("automation", opts.AutomationMode, "automation mode (manual|plan|auto|yolo)")
	dryRun := fs.Bool("dry-run", false, "print review prompt without calling harnesses")
	verbose := fs.Bool("verbose", false, "print individual harness reviews")
	if err := fs.Parse(args); err != nil {
		return dun.ExitUsageError
	}
	_ = *configPath

	docArgs := fs.Args()
	if len(docArgs) == 0 {
		docArgs = []string{"docs/helix/02-design/technical-designs/*.md"}
	}

	docs, err := loadReviewDocs(root, docArgs)
	if err != nil {
		fmt.Fprintf(stderr, "dun review failed: %v\n", err)
		return dun.ExitRuntimeError
	}
	if len(docs) == 0 {
		fmt.Fprintln(stderr, "dun review failed: no documents matched (provide paths or globs)")
		return dun.ExitRuntimeError
	}

	principles := ""
	if *principlesPath != "" {
		principles, err = readOptionalFile(root, *principlesPath)
		if err != nil {
			fmt.Fprintf(stderr, "dun review failed: %v\n", err)
			return dun.ExitRuntimeError
		}
	}

	resolvedHarnesses := resolveHarnessesForReview(*harnessesFlag)
	reviewCfg, err := dun.ParseQuorumFlags("", strings.Join(resolvedHarnesses, ","), false, false, "")
	if err != nil {
		fmt.Fprintf(stderr, "dun review failed: invalid harness list: %v\n", err)
		return dun.ExitUsageError
	}
	if len(reviewCfg.Harnesses) == 0 {
		fmt.Fprintln(stderr, "dun review failed: no harnesses configured")
		return dun.ExitUsageError
	}

	synth := *synthHarness
	if synth == "" {
		synth = reviewCfg.Harnesses[0]
	}

	modelOverrides := make(map[string]string)
	for harnessName, modelName := range opts.AgentModels {
		if modelName == "" {
			continue
		}
		modelOverrides[harnessName] = modelName
	}
	if *models != "" {
		parsed, parseErr := parseHarnessModelOverrides(*models)
		if parseErr != nil {
			fmt.Fprintf(stderr, "dun review failed: invalid models: %v\n", parseErr)
			return dun.ExitUsageError
		}
		for harnessName, modelName := range parsed {
			modelOverrides[harnessName] = modelName
		}
	}
	harnessModel = strings.TrimSpace(*model)
	if len(modelOverrides) == 0 {
		harnessModelOverrides = nil
	} else {
		harnessModelOverrides = modelOverrides
	}

	for _, doc := range docs {
		reviewPrompt := buildReviewPrompt(doc, principles, *principlesPath)
		if *dryRun {
			fmt.Fprintf(stdout, "--- REVIEW PROMPT (%s) ---\n", doc.Path)
			fmt.Fprintln(stdout, reviewPrompt)
			fmt.Fprintln(stdout, "--- END REVIEW PROMPT ---")
			continue
		}

		responses := make([]reviewResponse, len(reviewCfg.Harnesses))
		var wg sync.WaitGroup
		for i, harness := range reviewCfg.Harnesses {
			wg.Add(1)
			go func(idx int, name string) {
				defer wg.Done()
				resp, callErr := callHarness(name, reviewPrompt, *automation)
				responses[idx] = reviewResponse{Harness: name, Response: resp, Err: callErr}
			}(i, harness)
		}
		wg.Wait()

		var successful []reviewResponse
		var failures []reviewResponse
		for _, r := range responses {
			if r.Err != nil {
				failures = append(failures, r)
				fmt.Fprintf(stderr, "review harness failed (%s): %v\n", r.Harness, r.Err)
				continue
			}
			successful = append(successful, r)
		}
		if len(successful) == 0 {
			fmt.Fprintf(stderr, "dun review failed: all harnesses failed for %s\n", doc.Path)
			return dun.ExitRuntimeError
		}

		if *verbose {
			for _, r := range successful {
				fmt.Fprintf(stdout, "--- REVIEW (%s) ---\n", r.Harness)
				fmt.Fprintln(stdout, r.Response)
				fmt.Fprintln(stdout, "--- END REVIEW ---")
			}
		}

		synthPrompt := buildSynthesisPrompt(doc, principles, *principlesPath, successful, failures)
		synthResponse, err := callHarness(synth, synthPrompt, *automation)
		if err != nil {
			fmt.Fprintf(stderr, "dun review failed: synthesis harness failed (%s): %v\n", synth, err)
			return dun.ExitRuntimeError
		}

		fmt.Fprintf(stdout, "=== Review Summary: %s ===\n", doc.Path)
		fmt.Fprintln(stdout, synthResponse)
		fmt.Fprintln(stdout)
	}

	return dun.ExitSuccess
}

func loadReviewDocs(root string, patterns []string) ([]reviewDoc, error) {
	var paths []string
	seen := map[string]bool{}

	for _, pattern := range patterns {
		if hasGlob(pattern) {
			matches, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				return nil, err
			}
			for _, match := range matches {
				rel, err := filepath.Rel(root, match)
				if err != nil {
					return nil, err
				}
				rel = filepath.ToSlash(rel)
				if !seen[rel] {
					seen[rel] = true
					paths = append(paths, rel)
				}
			}
			continue
		}

		path := pattern
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if _, err := os.Stat(path); err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		if !seen[rel] {
			seen[rel] = true
			paths = append(paths, rel)
		}
	}

	sort.Strings(paths)

	docs := make([]reviewDoc, 0, len(paths))
	for _, rel := range paths {
		content, err := readOptionalFile(root, rel)
		if err != nil {
			return nil, err
		}
		docs = append(docs, reviewDoc{Path: rel, Content: content})
	}

	return docs, nil
}

func readOptionalFile(root string, path string) (string, error) {
	if path == "" {
		return "", nil
	}
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(root, path)
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func buildReviewPrompt(doc reviewDoc, principles string, principlesPath string) string {
	var b strings.Builder
	b.WriteString("You are an expert software engineer reviewing a plan before implementation.\n")
	b.WriteString("Your job is to assess clarity, completeness, feasibility, risk, and testability.\n")
	b.WriteString("Use the principles below as the primary evaluation rubric.\n\n")

	if principles != "" {
		b.WriteString("Principles (source: ")
		if principlesPath == "" {
			b.WriteString("unknown")
		} else {
			b.WriteString(principlesPath)
		}
		b.WriteString("):\n")
		b.WriteString("```")
		b.WriteString("\n")
		b.WriteString(principles)
		b.WriteString("\n```")
		b.WriteString("\n\n")
	}

	b.WriteString("Document to review:\n")
	b.WriteString("Path: ")
	b.WriteString(doc.Path)
	b.WriteString("\n\n")
	b.WriteString("```")
	b.WriteString("\n")
	b.WriteString(doc.Content)
	b.WriteString("\n```")
	b.WriteString("\n\n")

	b.WriteString("Review instructions:\n")
	b.WriteString("1) Summarize the intent in 2-3 sentences.\n")
	b.WriteString("2) List strengths that align with the principles.\n")
	b.WriteString("3) Identify major gaps or risks (blockers) and explain impact.\n")
	b.WriteString("4) Identify minor issues or polish opportunities.\n")
	b.WriteString("5) Call out missing test coverage, observability, or rollout steps.\n")
	b.WriteString("6) Note dependency or sequencing risks.\n")
	b.WriteString("7) Ask specific questions that must be answered before implementation.\n")
	b.WriteString("8) Provide a score from 0-10 with a one-line justification.\n")
	b.WriteString("9) Provide a final recommendation: approve, approve-with-changes, or block.\n\n")

	b.WriteString("Output format (markdown):\n")
	b.WriteString("- Summary\n")
	b.WriteString("- Strengths\n")
	b.WriteString("- Major Gaps/Risks\n")
	b.WriteString("- Minor Issues\n")
	b.WriteString("- Missing Tests/Validation\n")
	b.WriteString("- Dependency/Sequencing Concerns\n")
	b.WriteString("- Questions\n")
	b.WriteString("- Score (0-10) + Justification\n")
	b.WriteString("- Recommendation\n")

	return b.String()
}

func buildSynthesisPrompt(doc reviewDoc, principles string, principlesPath string, reviews []reviewResponse, failures []reviewResponse) string {
	if len(reviews) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("You are the lead reviewer. Synthesize multiple independent reviews into a single, high-quality review.\n")
	b.WriteString("Focus on consensus, highlight disagreements, and provide a final score and recommendation.\n")
	b.WriteString("Use the principles as the source of truth for the evaluation.\n\n")

	if principles != "" {
		b.WriteString("Principles (source: ")
		if principlesPath == "" {
			b.WriteString("unknown")
		} else {
			b.WriteString(principlesPath)
		}
		b.WriteString("):\n")
		b.WriteString("```")
		b.WriteString("\n")
		b.WriteString(principles)
		b.WriteString("\n```")
		b.WriteString("\n\n")
	}

	b.WriteString("Document under review:\n")
	b.WriteString("Path: ")
	b.WriteString(doc.Path)
	b.WriteString("\n\n")

	b.WriteString("Individual reviews:\n")
	for _, r := range reviews {
		b.WriteString("### ")
		b.WriteString(r.Harness)
		b.WriteString("\n")
		b.WriteString(r.Response)
		b.WriteString("\n\n")
	}

	if len(failures) > 0 {
		b.WriteString("Review failures:\n")
		for _, r := range failures {
			if r.Err == nil {
				continue
			}
			b.WriteString("- ")
			b.WriteString(r.Harness)
			b.WriteString(": ")
			b.WriteString(r.Err.Error())
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Synthesis instructions:\n")
	b.WriteString("1) Provide a concise summary and overall assessment.\n")
	b.WriteString("2) List the top 3 consensus strengths and top 3 consensus gaps.\n")
	b.WriteString("3) If reviewers disagreed, call it out explicitly and explain why.\n")
	b.WriteString("4) Provide a final score (0-10) and recommendation.\n")
	b.WriteString("5) Provide next steps required before implementation.\n\n")

	b.WriteString("Output format (markdown):\n")
	b.WriteString("- Summary\n")
	b.WriteString("- Consensus Strengths\n")
	b.WriteString("- Consensus Gaps/Risks\n")
	b.WriteString("- Disagreements\n")
	b.WriteString("- Score (0-10) + Justification\n")
	b.WriteString("- Recommendation\n")
	b.WriteString("- Next Steps\n")

	return b.String()
}

func hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}
