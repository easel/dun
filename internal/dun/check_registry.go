package dun

import "fmt"

// CheckConfig is a type-specific config payload for a check.
type CheckConfig interface{}

// CheckType defines a runnable check implementation.
type CheckType interface {
	Type() string
	Decode(spec Check) (CheckConfig, error)
	Run(root string, def CheckDefinition, cfg CheckConfig, opts Options, plugin Plugin) (CheckResult, error)
}

type checkHandler struct {
	typeName string
	decode   func(Check) (CheckConfig, error)
	run      func(string, CheckDefinition, CheckConfig, Options, Plugin) (CheckResult, error)
}

func (h checkHandler) Type() string {
	return h.typeName
}

func (h checkHandler) Decode(spec Check) (CheckConfig, error) {
	if h.decode == nil {
		return nil, nil
	}
	return h.decode(spec)
}

func (h checkHandler) Run(root string, def CheckDefinition, cfg CheckConfig, opts Options, plugin Plugin) (CheckResult, error) {
	if h.run == nil {
		return CheckResult{}, fmt.Errorf("check type %s missing run handler", h.typeName)
	}
	return h.run(root, def, cfg, opts, plugin)
}

var checkRegistry = map[string]CheckType{}

func RegisterCheckType(handler CheckType) {
	checkRegistry[handler.Type()] = handler
}

func LookupCheckType(name string) (CheckType, bool) {
	h, ok := checkRegistry[name]
	return h, ok
}
