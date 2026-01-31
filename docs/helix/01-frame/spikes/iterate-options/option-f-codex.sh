#!/bin/bash
# Option F: OpenAI Codex/GPT CLI
#
# Pros:
# - Wide ecosystem
# - Multiple model options (o3, GPT-4o, etc.)
# - Fresh context each iteration
#
# Cons:
# - Per-token pricing (no fixed fee like Claude Max)
# - Different tool/function calling syntax
# - Context limits vary by model
#
# Prerequisites:
#   npm install -g codex-cli  # or use openai directly
#   pip install openai

set -e

PROMPT='Here is a list of 10 words. Pick your favorite word, make it **bold**, and move it to OUTPUT.

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Respond with ONLY the updated INPUT/OUTPUT block, nothing else.'

echo "=== Option F: OpenAI/Codex CLI ==="

# Check for codex CLI
if command -v codex &> /dev/null; then
    echo "Running codex..."
    codex "$PROMPT"
elif command -v python3 &> /dev/null; then
    echo "Using Python SDK..."
    python3 << 'PYTHON'
import os
try:
    from openai import OpenAI
    client = OpenAI(api_key=os.environ.get("OPENAI_API_KEY", ""))
    response = client.chat.completions.create(
        model="gpt-4o",
        messages=[{"role": "user", "content": """Here is a list of 10 words. Pick your favorite word, make it **bold**, and move it to OUTPUT.

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Respond with ONLY the updated INPUT/OUTPUT block, nothing else."""}],
        max_tokens=256
    )
    print(response.choices[0].message.content)
    print(f"\nTokens: {response.usage.prompt_tokens} input, {response.usage.completion_tokens} output")
except ImportError:
    print("Install: pip install openai")
except Exception as e:
    print(f"Error: {e}")
PYTHON
else
    echo "Neither codex CLI nor python3 found"
    exit 1
fi

echo
echo "=== Done ==="
