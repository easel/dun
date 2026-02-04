package dun

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DoctorReport summarizes environment and project readiness.
type DoctorReport struct {
	Root      string
	Timestamp time.Time
	Harnesses []HarnessStatus
	Helpers   []HelperStatus
	Warnings  []string
	CachePath string
}

// HarnessStatus reports availability of a harness CLI.
type HarnessStatus struct {
	Name       string `json:"name"`
	Command    string `json:"command"`
	Available  bool   `json:"available"`
	Detail     string `json:"detail"`
	Live       bool   `json:"live"`
	Model      string `json:"model,omitempty"`
	LiveDetail string `json:"live_detail,omitempty"`
}

// HelperStatus reports availability of project helper tools.
type HelperStatus struct {
	Category  string
	Name      string
	Required  bool
	Available bool
	Detail    string
}

// RunDoctor performs environment and project checks and updates the harness cache.
func RunDoctor(root string) (DoctorReport, error) {
	report := DoctorReport{
		Root:      root,
		Timestamp: time.Now(),
	}

	report.Harnesses = checkHarnesses()
	report.Helpers = checkProjectHelpers(root)

	path, err := HarnessCachePath()
	if err != nil {
		return report, err
	}
	report.CachePath = path

	cache := HarnessCache{LastCheck: report.Timestamp, Harnesses: report.Harnesses}
	if err := cache.SaveTo(path); err != nil {
		return report, err
	}

	return report, nil
}

var harnessLivenessFn = checkHarnessLiveness

func checkHarnesses() []HarnessStatus {
	names := DefaultRegistry.List()
	sort.Strings(names)
	statuses := make([]HarnessStatus, 0, len(names))
	for _, name := range names {
		if name == "mock" {
			continue
		}
		cmd := defaultHarnessCommand(name)
		path, err := exec.LookPath(cmd)
		status := HarnessStatus{Name: name, Command: cmd}
		if err != nil {
			status.Available = false
			status.Detail = "command not found"
			statuses = append(statuses, status)
			continue
		}
		status.Available = true
		status.Detail = "found " + path
		live, model, detail := harnessLivenessFn(name)
		status.Live = live
		status.Model = model
		status.LiveDetail = detail
		statuses = append(statuses, status)
	}
	return statuses
}

func defaultHarnessCommand(name string) string {
	switch name {
	case "claude", "gemini", "codex", "opencode":
		return name
	default:
		return name
	}
}

const doctorHarnessTimeout = 30 * time.Second

const doctorPrompt = `Reply with JSON on one line: {"ok":true,"model":"<model name>"}. If unknown, use "unknown".`

type doctorPing struct {
	OK    bool   `json:"ok"`
	Model string `json:"model"`
}

func checkHarnessLiveness(name string) (bool, string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), doctorHarnessTimeout)
	defer cancel()

	harness, err := DefaultRegistry.Get(name, HarnessConfig{
		Name:           name,
		AutomationMode: AutomationAuto,
		Timeout:        doctorHarnessTimeout,
	})
	if err != nil {
		return false, "", err.Error()
	}

	response, err := harness.Execute(ctx, doctorPrompt)
	if err != nil {
		return false, "", err.Error()
	}

	model, detail := parseDoctorResponse(response)
	return true, model, detail
}

func parseDoctorResponse(response string) (string, string) {
	candidate := extractJSON(response)
	if candidate != "" {
		var ping doctorPing
		if err := json.Unmarshal([]byte(candidate), &ping); err == nil {
			model := strings.TrimSpace(ping.Model)
			if model == "" {
				model = "unknown"
			}
			return model, ""
		}
	}
	model := extractModelHint(response)
	if model != "" {
		return model, "non-json response"
	}
	return "", "unexpected response"
}

func extractJSON(response string) string {
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return response[start : end+1]
}

func extractModelHint(response string) string {
	lower := strings.ToLower(response)
	for _, key := range []string{"model:", "model="} {
		if idx := strings.Index(lower, key); idx != -1 {
			fragment := response[idx+len(key):]
			fragment = strings.TrimSpace(fragment)
			if fragment == "" {
				return ""
			}
			fields := strings.Fields(fragment)
			if len(fields) == 0 {
				return ""
			}
			return strings.Trim(fields[0], "\"',.")
		}
	}
	return ""
}

func checkProjectHelpers(root string) []HelperStatus {
	var helpers []HelperStatus
	if fileExists(filepath.Join(root, "go.mod")) {
		helpers = append(helpers, checkGoHelpers()...)
	}
	if fileExists(filepath.Join(root, ".git")) {
		helpers = append(helpers, checkGitHelpers(root)...)
	}
	if fileExists(filepath.Join(root, ".beads")) {
		helpers = append(helpers, checkBeadsHelpers()...)
	}
	return helpers
}

func checkGoHelpers() []HelperStatus {
	var helpers []HelperStatus
	goPath, goErr := exec.LookPath("go")
	goAvailable := goErr == nil
	goDetail := "command not found"
	if goAvailable {
		goDetail = "found " + goPath
	}
	helpers = append(helpers, HelperStatus{
		Category:  "go",
		Name:      "go",
		Required:  true,
		Available: goAvailable,
		Detail:    goDetail,
	})

	coverAvailable := false
	coverDetail := "go not available"
	if goAvailable {
		coverDetail = "go tool cover not available"
		if err := runGoToolCoverCheck(); err == nil {
			coverAvailable = true
			coverDetail = "ok"
		} else {
			coverDetail = err.Error()
		}
	}
	helpers = append(helpers, HelperStatus{
		Category:  "go",
		Name:      "go tool cover",
		Required:  true,
		Available: coverAvailable,
		Detail:    coverDetail,
	})

	helpers = append(helpers,
		toolHelper("go", "staticcheck", false),
		toolHelper("go", "govulncheck", false),
		toolHelper("go", "gosec", false),
	)

	return helpers
}

func runGoToolCoverCheck() error {
	cmd := exec.Command("go", "tool", "cover", "-h")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func toolHelper(category string, name string, required bool) HelperStatus {
	path, err := exec.LookPath(name)
	if err != nil {
		return HelperStatus{
			Category:  category,
			Name:      name,
			Required:  required,
			Available: false,
			Detail:    "command not found",
		}
	}
	return HelperStatus{
		Category:  category,
		Name:      name,
		Required:  required,
		Available: true,
		Detail:    "found " + path,
	}
}

func checkGitHelpers(root string) []HelperStatus {
	var helpers []HelperStatus
	helpers = append(helpers, toolHelper("git", "git", true))
	hook, err := detectHookTool(root)
	if err != nil {
		helpers = append(helpers, HelperStatus{
			Category:  "git",
			Name:      "hook tool",
			Required:  false,
			Available: false,
			Detail:    err.Error(),
		})
		return helpers
	}
	if hook.Name == "" {
		return helpers
	}
	status := HelperStatus{
		Category:  "git",
		Name:      hook.Name,
		Required:  false,
		Available: hook.Installed,
	}
	if hook.Installed {
		status.Detail = "configured"
	} else {
		status.Detail = hook.InstallHint
	}
	helpers = append(helpers, status)
	return helpers
}

func checkBeadsHelpers() []HelperStatus {
	status := toolHelper("beads", "bd", false)
	if !status.Available {
		status.Detail = "beads config detected; install bd"
	}
	return []HelperStatus{status}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FormatDoctorReport renders a human-readable report for CLI output.
func FormatDoctorReport(report DoctorReport) string {
	var b strings.Builder
	b.WriteString("dun doctor\n")
	if report.Root != "" {
		b.WriteString("root: ")
		b.WriteString(report.Root)
		b.WriteString("\n")
	}
	if len(report.Harnesses) > 0 {
		b.WriteString("\nHarnesses:\n")
		for _, harness := range report.Harnesses {
			status := "missing"
			if harness.Available {
				status = "ok"
			}
			b.WriteString(fmt.Sprintf("- %s: %s", harness.Name, status))
			if harness.Detail != "" {
				b.WriteString(" (")
				b.WriteString(harness.Detail)
				b.WriteString(")")
			}
			if harness.Available {
				liveStatus := "fail"
				if harness.Live {
					liveStatus = "ok"
				}
				b.WriteString("; live: ")
				b.WriteString(liveStatus)
				if harness.Model != "" {
					b.WriteString(" (model: ")
					b.WriteString(harness.Model)
					b.WriteString(")")
				} else if harness.LiveDetail != "" {
					b.WriteString(" (")
					b.WriteString(harness.LiveDetail)
					b.WriteString(")")
				}
			}
			b.WriteString("\n")
		}
	}

	if len(report.Helpers) > 0 {
		b.WriteString("\nHelpers:\n")
		groups := groupHelpers(report.Helpers)
		for _, group := range groups {
			b.WriteString(fmt.Sprintf("%s:\n", group.Category))
			for _, helper := range group.Helpers {
				status := helperStatusLabel(helper)
				b.WriteString(fmt.Sprintf("- %s: %s", helper.Name, status))
				if helper.Detail != "" {
					b.WriteString(" (")
					b.WriteString(helper.Detail)
					b.WriteString(")")
				}
				b.WriteString("\n")
			}
		}
	}

	if len(report.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warn := range report.Warnings {
			b.WriteString("- ")
			b.WriteString(warn)
			b.WriteString("\n")
		}
	}
	if report.CachePath != "" {
		b.WriteString("\nCache: updated ")
		b.WriteString(report.CachePath)
		b.WriteString("\n")
	}

	return b.String()
}

type helperGroup struct {
	Category string
	Helpers  []HelperStatus
}

func groupHelpers(helpers []HelperStatus) []helperGroup {
	byCat := make(map[string][]HelperStatus)
	order := make([]string, 0)
	for _, helper := range helpers {
		if _, ok := byCat[helper.Category]; !ok {
			order = append(order, helper.Category)
		}
		byCat[helper.Category] = append(byCat[helper.Category], helper)
	}
	sort.Strings(order)
	groups := make([]helperGroup, 0, len(order))
	for _, category := range order {
		entries := byCat[category]
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})
		groups = append(groups, helperGroup{Category: category, Helpers: entries})
	}
	return groups
}

func helperStatusLabel(helper HelperStatus) string {
	if helper.Available {
		return "ok"
	}
	if helper.Required {
		return "fail"
	}
	return "warn"
}
