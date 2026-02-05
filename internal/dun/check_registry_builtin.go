package dun

import "fmt"

func init() {
	RegisterCheckType(checkHandler{
		typeName: "rule-set",
		decode: func(spec Check) (CheckConfig, error) {
			return RuleSetConfig{Rules: spec.Rules}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			config, ok := cfg.(RuleSetConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("rule-set config missing")
			}
			return runRuleSet(root, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "gates",
		decode: func(spec Check) (CheckConfig, error) {
			return GateConfig{GateFiles: spec.GateFiles}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, plugin Plugin) (CheckResult, error) {
			config, ok := cfg.(GateConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("gates config missing")
			}
			return runGateCheck(root, plugin, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "state-rules",
		decode: func(spec Check) (CheckConfig, error) {
			return StateRulesConfig{StateRules: spec.StateRules}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, plugin Plugin) (CheckResult, error) {
			config, ok := cfg.(StateRulesConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("state-rules config missing")
			}
			return runStateRules(root, plugin, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "agent",
		decode: func(spec Check) (CheckConfig, error) {
			return AgentCheckConfig{Prompt: spec.Prompt, Inputs: spec.Inputs, ResponseSchema: spec.ResponseSchema}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, opts Options, plugin Plugin) (CheckResult, error) {
			config, ok := cfg.(AgentCheckConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("agent config missing")
			}
			return runAgentCheck(root, plugin, def, config, opts)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "git-status",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runGitStatusCheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "hook-check",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runHookCheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "command",
		decode: func(spec Check) (CheckConfig, error) {
			return CommandConfig{
				Command:      spec.Command,
				Parser:       spec.Parser,
				SuccessExit:  spec.SuccessExit,
				WarnExits:    spec.WarnExits,
				Timeout:      spec.Timeout,
				Shell:        spec.Shell,
				Env:          spec.Env,
				IssuePath:    spec.IssuePath,
				IssuePattern: spec.IssuePattern,
				IssueFields:  spec.IssueFields,
			}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			config, ok := cfg.(CommandConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("command config missing")
			}
			return runCommandCheck(root, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "go-test",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runGoTestCheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "go-coverage",
		decode: func(spec Check) (CheckConfig, error) {
			return GoCoverageConfig{Rules: spec.Rules}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, opts Options, _ Plugin) (CheckResult, error) {
			config, ok := cfg.(GoCoverageConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("go-coverage config missing")
			}
			return runGoCoverageCheck(root, def, config, opts)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "go-vet",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runGoVetCheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "go-staticcheck",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runGoStaticcheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "beads-ready",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runBeadsReadyCheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "beads-critical-path",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runBeadsCriticalPathCheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "beads-suggest",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runBeadsSuggestCheck(root, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "spec-binding",
		decode: func(spec Check) (CheckConfig, error) {
			return SpecBindingConfig{Bindings: spec.Bindings, BindingRules: spec.BindingRules}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			config, ok := cfg.(SpecBindingConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("spec-binding config missing")
			}
			return runSpecBindingCheck(root, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "change-cascade",
		decode: func(spec Check) (CheckConfig, error) {
			return extractCascadeConfig(spec), nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			config, ok := cfg.(ChangeCascadeConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("change-cascade config missing")
			}
			return runChangeCascadeCheck(root, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "integration-contract",
		decode: func(spec Check) (CheckConfig, error) {
			return IntegrationContractConfig{Contracts: spec.Contracts, ContractRules: spec.ContractRules}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			config, ok := cfg.(IntegrationContractConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("integration-contract config missing")
			}
			return runIntegrationContractCheck(root, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "conflict-detection",
		decode: func(spec Check) (CheckConfig, error) {
			return ConflictDetectionConfig{Tracking: spec.Tracking, ConflictRules: spec.ConflictRules}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			config, ok := cfg.(ConflictDetectionConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("conflict-detection config missing")
			}
			return runConflictDetectionCheck(root, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "agent-rule-injection",
		decode: func(spec Check) (CheckConfig, error) {
			return AgentRuleInjectionConfig{
				BasePrompt:   spec.BasePrompt,
				InjectRules:  spec.InjectRules,
				EnforceRules: spec.EnforceRules,
			}, nil
		},
		run: func(root string, def CheckDefinition, cfg CheckConfig, _ Options, plugin Plugin) (CheckResult, error) {
			config, ok := cfg.(AgentRuleInjectionConfig)
			if !ok {
				return CheckResult{}, fmt.Errorf("agent-rule-injection config missing")
			}
			return runAgentRuleInjectionCheck(root, plugin, def, config)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "doc-dag",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, plugin Plugin) (CheckResult, error) {
			return runDocDagCheck(root, plugin, def)
		},
	})

	RegisterCheckType(checkHandler{
		typeName: "self-test",
		run: func(root string, def CheckDefinition, _ CheckConfig, _ Options, _ Plugin) (CheckResult, error) {
			return runSelfTestCheck(root, def)
		},
	})
}
