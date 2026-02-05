package dun

// Typed config structures for check types.

type RuleSetConfig struct {
	Rules []Rule
}

type GateConfig struct {
	GateFiles []string
}

type StateRulesConfig struct {
	StateRules string
}

type AgentCheckConfig struct {
	Prompt         string
	Inputs         []string
	ResponseSchema string
}

type CommandConfig struct {
	Command      string
	Parser       string
	SuccessExit  int
	WarnExits    []int
	Timeout      string
	Shell        string
	Env          map[string]string
	IssuePath    string
	IssuePattern string
	IssueFields  IssueFieldMap
}

type GoCoverageConfig struct {
	Rules []Rule
}

type SpecBindingConfig struct {
	Bindings     SpecBindings
	BindingRules []BindingRule
}

type IntegrationContractConfig struct {
	Contracts     ContractsConfig
	ContractRules []ContractRule
}

type ConflictDetectionConfig struct {
	Tracking      TrackingConfig
	ConflictRules []ConflictRule
}

type AgentRuleInjectionConfig struct {
	BasePrompt   string
	InjectRules  []InjectRule
	EnforceRules []EnforceRule
}
