#!/usr/bin/env bash
set -euo pipefail

# Generate all devctl demo GIFs using VHS

cd "$(dirname "$0")"

if ! command -v vhs >/dev/null 2>&1; then
    echo "Error: vhs not found. Install with: brew install vhs"
    echo "Or see: https://github.com/charmbracelet/vhs#installation"
    exit 1
fi

echo "Generating devctl demo GIFs..."
echo

for tape in *.tape; do
    echo "→ Processing $tape..."
    vhs "$tape"
    echo "  ✓ Generated ${tape%.tape}.gif"
    echo
done

echo "All GIFs generated successfully!"
echo
echo "GIFs are in:"
echo "  $(pwd)/"
echo
echo "To use in documentation:"
echo "  ![devctl demo](vhs/01-cli-workflow.gif)"

