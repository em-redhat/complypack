#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Validates the OpenPackage manifest for generating-gemara-policies skill.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="${SCRIPT_DIR}/openpackage.yml"

# Check manifest exists
if [[ ! -f "$MANIFEST" ]]; then
    echo "ERROR: openpackage.yml not found"
    exit 1
fi

# Validate YAML syntax
if ! python3 -c "import yaml; yaml.safe_load(open('$MANIFEST'))" 2>/dev/null; then
    echo "ERROR: Invalid YAML syntax in openpackage.yml"
    exit 1
fi

# Check required fields
required_fields=("name" "version" "description" "author" "license")
for field in "${required_fields[@]}"; do
    if ! grep -q "^${field}:" "$MANIFEST"; then
        echo "ERROR: Missing required field: $field"
        exit 1
    fi
done

# Validate name matches directory
manifest_name=$(grep "^name:" "$MANIFEST" | awk '{print $2}')
expected_name="generating-gemara-policies"
if [[ "$manifest_name" != "$expected_name" ]]; then
    echo "ERROR: Manifest name '$manifest_name' doesn't match directory '$expected_name'"
    exit 1
fi

# Check skill file exists
if [[ ! -f "${SCRIPT_DIR}/SKILL.md" ]]; then
    echo "ERROR: SKILL.md not found"
    exit 1
fi

echo "✓ OpenPackage manifest is valid"
echo "✓ Name: $manifest_name"
echo "✓ Version: $(grep '^version:' "$MANIFEST" | awk '{print $2}')"
echo "✓ SKILL.md present"
