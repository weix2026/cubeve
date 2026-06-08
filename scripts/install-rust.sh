#!/bin/bash
# Install Rust toolchain offline-compatible
set -e

if ! command -v rustc &> /dev/null; then
    echo "Installing Rust via apt..."
    apt-get update && apt-get install -y rustc cargo
fi

echo "Rust installed: $(rustc --version)"
echo "Cargo installed: $(cargo --version)"

# Create vendor directory for offline builds if needed
if [ "$1" == "vendor" ]; then
    echo "Creating vendor cache..."
    cd /root/.openclaw/workspace
    mkdir -p .cargo
    cat > .cargo/config.toml << 'EOF'
[source.crates-io]
replace-with = "vendored-sources"

[source.vendored-sources]
directory = "vendor"
EOF
    echo "Vendor config created. Run 'cargo vendor' when network available."
fi
