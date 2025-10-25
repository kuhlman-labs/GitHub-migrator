#!/bin/bash

# Script to download git-sizer binaries for embedding
# These binaries will be embedded into the Go application

set -e

# git-sizer version to download
VERSION="${GIT_SIZER_VERSION:-1.5.0}"
BASE_URL="https://github.com/github/git-sizer/releases/download/v${VERSION}"

# Directory to store binaries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/../internal/embedded/bin"

# Create bin directory if it doesn't exist
mkdir -p "${BIN_DIR}"

echo "Downloading git-sizer v${VERSION} binaries..."

# Function to download and extract binary
download_binary() {
    local platform=$1
    local arch=$2
    local archive_name=$3
    local binary_name=$4
    local output_name=$5

    echo "Downloading ${platform}/${arch}..."
    
    local url="${BASE_URL}/${archive_name}"
    local temp_dir=$(mktemp -d)
    
    # Download archive with proper redirect following and headers
    if command -v curl >/dev/null 2>&1; then
        curl -L -H "Accept: application/octet-stream" -o "${temp_dir}/archive" "${url}" || {
            echo "  ✗ Download failed for ${archive_name}"
            rm -rf "${temp_dir}"
            return 1
        }
    elif command -v wget >/dev/null 2>&1; then
        wget --header="Accept: application/octet-stream" -O "${temp_dir}/archive" "${url}" || {
            echo "  ✗ Download failed for ${archive_name}"
            rm -rf "${temp_dir}"
            return 1
        }
    else
        echo "Error: Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    # Verify download size (should be at least 1MB)
    local file_size=$(stat -f%z "${temp_dir}/archive" 2>/dev/null || stat -c%s "${temp_dir}/archive" 2>/dev/null || echo "0")
    if [ "$file_size" -lt 1000000 ]; then
        echo "  ✗ Download failed: file too small (${file_size} bytes, expected >1MB)"
        echo "  This usually means the URL is incorrect or the release doesn't exist"
        rm -rf "${temp_dir}"
        return 1
    fi
    
    # Extract binary
    cd "${temp_dir}"
    if [[ "${archive_name}" == *.zip ]]; then
        unzip -q archive
    else
        tar -xzf archive
    fi
    
    # Copy binary to bin directory
    if [ -f "${binary_name}" ]; then
        cp "${binary_name}" "${BIN_DIR}/${output_name}"
        chmod +x "${BIN_DIR}/${output_name}"
        echo "  ✓ Saved to ${output_name} ($(du -h "${BIN_DIR}/${output_name}" | cut -f1))"
    else
        echo "  ✗ Binary not found in archive: ${binary_name}"
        echo "  Contents of archive:"
        ls -la
        rm -rf "${temp_dir}"
        return 1
    fi
    
    # Cleanup
    rm -rf "${temp_dir}"
}

# Download binaries for each platform
# Track successes and failures
FAILED=()
SUCCEEDED=()

download_and_track() {
    if download_binary "$@"; then
        SUCCEEDED+=("$1/$2")
    else
        FAILED+=("$1/$2")
    fi
}

download_and_track "linux" "amd64" "git-sizer-${VERSION}-linux-amd64.zip" "git-sizer" "git-sizer-linux-amd64"
download_and_track "darwin" "amd64" "git-sizer-${VERSION}-darwin-amd64.zip" "git-sizer" "git-sizer-darwin-amd64"
download_and_track "darwin" "arm64" "git-sizer-${VERSION}-darwin-arm64.zip" "git-sizer" "git-sizer-darwin-arm64"
download_and_track "windows" "amd64" "git-sizer-${VERSION}-windows-amd64.zip" "git-sizer.exe" "git-sizer-windows-amd64.exe"

# Note: linux-arm64 is not available in git-sizer releases
# Create a placeholder that will trigger fallback to system git-sizer
touch "${BIN_DIR}/git-sizer-linux-arm64"
echo "Note: git-sizer does not provide linux-arm64 builds. System git-sizer will be used on that platform."

echo ""
echo "Download Summary:"
echo "  ✓ Succeeded: ${#SUCCEEDED[@]}"
echo "  ✗ Failed: ${#FAILED[@]}"

if [ ${#SUCCEEDED[@]} -gt 0 ]; then
    echo ""
    echo "Successfully downloaded binaries:"
    ls -lh "${BIN_DIR}"/git-sizer-* 2>/dev/null || echo "  (no binaries found)"
fi

if [ ${#FAILED[@]} -gt 0 ]; then
    echo ""
    echo "Failed downloads:"
    for platform in "${FAILED[@]}"; do
        echo "  - ${platform}"
    done
    echo ""
    echo "Note: You can still build for platforms that succeeded."
    echo "The application will fall back to system git-sizer for missing platforms."
fi

echo ""
echo "Note: These binaries are embedded into the Go application during build."
echo "They will be extracted at runtime to a temporary directory."

# Exit with error if ALL downloads failed
if [ ${#SUCCEEDED[@]} -eq 0 ]; then
    echo ""
    echo "ERROR: All downloads failed. Please check your internet connection and try again."
    echo "Or install git-sizer system-wide: go install github.com/github/git-sizer@latest"
    exit 1
fi

