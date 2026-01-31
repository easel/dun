#!/bin/bash
# Option E: Gemini CLI
#
# Pros:
# - 1M-2M context window (largest available)
# - Fresh context each iteration
# - Simple CLI interface
#
# Cons:
# - Different API/capabilities than Claude
# - May need different prompt tuning
# - Tool use differences
#
# Prerequisites:
#   npm install -g @anthropic-ai/gemini-cli  # or
#   pip install google-generativeai

set -e

PROMPT='Here is a list of 10 words. Pick your favorite word, make it **bold**, and move it to OUTPUT.

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Respond with ONLY the updated INPUT/OUTPUT block, nothing else.'

echo "=== Option E: Gemini CLI ==="

# Check for gemini CLI
if command -v gemini &> /dev/null; then
    echo "Running gemini..."
    gemini "$PROMPT"
elif command -v python3 &> /dev/null; then
    echo "Using Python SDK..."
    python3 << 'PYTHON'
import os
try:
    import google.generativeai as genai
    genai.configure(api_key=os.environ.get("GOOGLE_API_KEY", ""))
    model = genai.GenerativeModel("gemini-1.5-flash")
    response = model.generate_content("""Here is a list of 10 words. Pick your favorite word, make it **bold**, and move it to OUTPUT.

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Respond with ONLY the updated INPUT/OUTPUT block, nothing else.""")
    print(response.text)
except ImportError:
    print("Install: pip install google-generativeai")
except Exception as e:
    print(f"Error: {e}")
PYTHON
else
    echo "Neither gemini CLI nor python3 found"
    exit 1
fi

echo
echo "=== Done ==="
