#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
OUTPUT_BINARY="steam-download-tool"

echo "=== Steam Workshop Download Tool - Build Script ==="
echo ""

# Step 1: Build frontend
echo "--- [1/3] Building Vue frontend ---"
cd "$PROJECT_ROOT/web"

if [ ! -d "node_modules" ]; then
    echo "Installing npm dependencies..."
    npm install
fi

echo "Running vite build..."
npx vite build --outDir dist

# Fix: Go's embed package ignores files/dirs starting with _ or .
# Rename _-prefixed files and update all references accordingly.
echo "Fixing _-prefixed files for Go embed compatibility..."
find dist -name '_*' -type f | while IFS= read -r oldpath; do
    dir=$(dirname "$oldpath")
    oldname=$(basename "$oldpath")
    newname=$(echo "$oldname" | sed 's/^_//')
    newpath="$dir/$newname"

    if [ "$oldname" != "$newname" ]; then
        echo "  Renaming: $oldpath -> $newpath"
        mv "$oldpath" "$newpath"

        # Update references in all files under dist/
        find dist -type f \( -name '*.js' -o -name '*.css' -o -name '*.html' \) \
            -exec sed -i "s|$oldname|$newname|g" {} +
    fi
done

echo "Frontend built: web/dist/"

# Step 2: Build Go backend
echo ""
echo "--- [2/3] Building Go backend (CGO disabled) ---"
cd "$PROJECT_ROOT"

# Ensure dependencies are downloaded
go mod tidy

CGO_ENABLED=0 go build -ldflags="-s -w" -o "$OUTPUT_BINARY" .
echo "Binary built: $OUTPUT_BINARY"

# Step 3: Set permissions
chmod +x "$OUTPUT_BINARY"

echo ""
echo "=== Build complete ==="
echo "Binary: $PROJECT_ROOT/$OUTPUT_BINARY"
echo "Run with: ./$OUTPUT_BINARY"
echo ""
echo "Environment variables:"
echo "  PORT=8086                   HTTP server port"
echo "  JWT_SECRET=...              JWT signing secret"
echo "  AES_KEY=...                 AES-256 encryption key for Steam passwords (required)"
echo "  GITHUB_CLIENT_ID=...        GitHub OAuth client ID"
echo "  GITHUB_CLIENT_SECRET=...    GitHub OAuth client secret"
echo "  MAX_WORKERS=2               Max concurrent downloads"
echo "  FILE_TTL_HOURS=72           Hours before files expire"
echo "  DATABASE_PATH=./data/steam-download.db"
echo "  DEPOT_DOWNLOADER_PATH=./DepotDownloader"
