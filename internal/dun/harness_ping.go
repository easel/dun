package dun

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const HarnessPingPrompt = `Reply with JSON on one line: {"ok":true,"model":"<model name>"}. If unknown, use "unknown".`

const defaultHarnessPingTimeout = 30 * time.Second

// HarnessPingResult captures the outcome of a harness liveness ping.
type HarnessPingResult struct {
	Live     bool
	Model    string
	Detail   string
	Response string
}

// PingHarness sends a short liveness prompt to the harness and parses the response.
func PingHarness(ctx context.Context, name string, config HarnessConfig) (HarnessPingResult, error) {
	if config.AutomationMode == "" {
		config.AutomationMode = AutomationAuto
	}
	if config.Timeout == 0 {
		config.Timeout = defaultHarnessPingTimeout
	}
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	harness, err := DefaultRegistry.Get(name, config)
	if err != nil {
		return HarnessPingResult{Live: false, Detail: err.Error()}, err
	}
	if !harness.SupportsAutomation(config.AutomationMode) {
		err := fmt.Errorf("harness %s does not support automation mode %s", name, config.AutomationMode)
		return HarnessPingResult{Live: false, Detail: err.Error()}, err
	}

	response, err := harness.Execute(ctx, HarnessPingPrompt)
	if err != nil {
		return HarnessPingResult{Live: false, Detail: err.Error()}, err
	}

	model, detail := parseHarnessPingResponse(response)
	return HarnessPingResult{Live: true, Model: model, Detail: detail, Response: response}, nil
}

func parseHarnessPingResponse(response string) (string, string) {
	candidate := extractJSON(response)
	if candidate != "" {
		var ping struct {
			OK    bool   `json:"ok"`
			Model string `json:"model"`
		}
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
