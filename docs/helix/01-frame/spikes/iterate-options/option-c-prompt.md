# Option C: Claude Code + Task Tool

Run this prompt inside a Claude Code session to test the Task tool approach.

## The Prompt

```
I want to test using the Task tool for autonomous work. Here's a simple task:

Pick your favorite word from this list, make it **bold**, and move it to OUTPUT:

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Please spawn a "general-purpose" agent using the Task tool to:
1. Read the INPUT line
2. Pick a favorite word
3. Format it as bold (**word**)
4. Return the result

After the agent completes, show me the final result.
```

## Expected Behavior

Claude Code should:
1. Use the Task tool to spawn an agent
2. The agent picks a word and returns it
3. Claude Code shows the result

## Pros
- Agents can work in parallel
- Shared session context (if desired)
- Claude Code handles complexity
- Can spawn specialized agents (coder, researcher, etc.)

## Cons
- Requires running inside Claude Code session
- More complex orchestration
- Context accumulation (may cause drift)
- Harder to automate from external scripts

## Key Question

Can we get the Task tool result back programmatically?
- Task tool returns agent output to Claude Code
- Claude Code would need to format it for dun to consume
- This creates a "Claude Code as orchestrator" pattern
