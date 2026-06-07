#!/bin/bash
# Validate all CubeVE deployment scripts
# Usage: ./scripts/validate-scripts.sh

set -e

SCRIPTS_DIR="./scripts"
FAILED=0
PASSED=0

echo "Validating CubeVE Deployment Scripts"
echo "===================================="
echo ""

validate_script() {
    local script=$1
    local name=$(basename "$script")
    
    echo -n "Validating $name... "
    
    # Check if file exists and is executable
    if [ ! -f "$script" ]; then
        echo "❌ (not found)"
        FAILED=$((FAILED + 1))
        return
    fi
    
    # Check syntax with bash
    if bash -n "$script" 2>/dev/null; then
        echo "✅ (syntax OK)"
        PASSED=$((PASSED + 1))
    else
        echo "❌ (syntax error)"
        FAILED=$((FAILED + 1))
    fi
}

# Validate all scripts
for script in "$SCRIPTS_DIR"/*.sh; do
    if [ -f "$script" ]; then
        validate_script "$script"
    fi
done

echo ""
echo "===================================="
echo "Results: $PASSED passed, $FAILED failed"

if [ $FAILED -gt 0 ]; then
    exit 1
fi

echo "All scripts validated successfully!"
