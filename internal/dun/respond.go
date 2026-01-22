package dun

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func Respond(id string, reader io.Reader) (CheckResult, error) {
	if strings.TrimSpace(id) == "" {
		return CheckResult{}, fmt.Errorf("response id is required")
	}
	var resp AgentResponse
	dec := json.NewDecoder(reader)
	if err := dec.Decode(&resp); err != nil {
		return CheckResult{}, fmt.Errorf("parse response: %w", err)
	}
	if resp.Status == "" || resp.Signal == "" {
		return CheckResult{}, fmt.Errorf("agent response missing required fields")
	}
	return CheckResult{
		ID:     id,
		Status: resp.Status,
		Signal: resp.Signal,
		Detail: resp.Detail,
		Next:   resp.Next,
		Issues: resp.Issues,
	}, nil
}
