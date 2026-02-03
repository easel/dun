## Tools
<!-- DUN:BEGIN -->
- **dun**: Development quality checker with autonomous loop support

  Quick commands:
  - `dun check` - Run all quality checks
  - `dun check --prompt` - Get work list as a prompt (pick ONE task, complete it, exit)
  - `dun loop --harness claude` - Run autonomous loop with Claude
  - `dun loop --harness gemini` - Run autonomous loop with Gemini
  - `dun loop --harness opencode` - Run autonomous loop with OpenCode
  - `dun loop --harness pi` - Run autonomous loop with Pi
  - `dun help` - Full documentation

  Autonomous iteration pattern:
  1. Run `dun check --prompt` to see available work
  2. Pick ONE task with highest impact
  3. Complete that task fully (edit files, run tests)
  4. Exit - the loop will call you again for the next task
<!-- DUN:END -->
