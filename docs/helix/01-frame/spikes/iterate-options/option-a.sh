#!/bin/bash
# Option A: Pure bash loop (Ralph pattern)
#
# Pros:
# - Simplest implementation
# - Fresh context each iteration (no drift)
# - Works with Claude Max fixed-fee
# - No Go code changes needed
#
# Cons:
# - Can't easily pass structured data back to dun
# - Limited error handling
# - No programmatic control of Claude behavior

set -e

PROMPT='Here is a list of 10 words. Pick your favorite word, make it **bold**, and move it to OUTPUT.

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Respond with ONLY the updated INPUT/OUTPUT block, nothing else.'

echo "=== Option A: Bash Loop ==="
echo "Running claude with prompt..."
echo

# Single iteration (remove 'exit' for loop)
# Use stdin to avoid OS argument length limits for large prompts.
# For yolo: add --dangerously-skip-permissions
printf "%s" "$PROMPT" | claude --print --input-format text --output-format text 2>/dev/null

echo
echo "=== Done ==="

# For actual loop:
# MAX_ITERATIONS=10
# for i in $(seq 1 $MAX_ITERATIONS); do
#     echo "Iteration $i..."
#     RESULT=$(claude -p "$PROMPT" --output-format text 2>/dev/null)
#     echo "$RESULT"
#     # Parse result, check exit conditions, etc.
# done
