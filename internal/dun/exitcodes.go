package dun

// Exit codes for dun CLI
const (
	ExitSuccess      = 0 // All checks pass
	ExitCheckFailed  = 1 // One or more checks failed
	ExitConfigError  = 2 // Configuration error
	ExitRuntimeError = 3 // Runtime error (command not found, etc)
	ExitUsageError   = 4 // Usage error (bad flags, missing args)
)
