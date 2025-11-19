# Embedded Binaries

This package embeds external binary dependencies (like `git-sizer`) into the Go application, allowing for single-binary distribution.

## How It Works

1. **Download Phase**: During the build process, the `scripts/download-git-sizer.sh` script downloads pre-compiled `git-sizer` binaries for all supported platforms
2. **Embed Phase**: Go's `//go:embed` directive includes these binaries in the compiled application
3. **Runtime Extraction**: On first use, the appropriate binary for the current platform is extracted to a temporary directory
4. **Execution**: The application uses the extracted binary for all operations

## Supported Platforms

- Linux: amd64, arm64
- macOS (Darwin): amd64, arm64
- Windows: amd64

## Build Process

### Local Build

```bash
# Download binaries and build
make build

# Or manually:
./scripts/download-git-sizer.sh
go build -o bin/github-migrator-server cmd/server/main.go
```

### Docker Build

The Dockerfile automatically downloads binaries during the build stage:

```bash
docker build -t github-migrator .
```

### CI/CD

Include the binary download step in your CI/CD pipeline:

```yaml
- name: Download embedded binaries
  run: ./scripts/download-git-sizer.sh
  
- name: Build
  run: go build -o bin/server cmd/server/main.go
```

## Development

For development without downloading binaries, the application can fall back to using system-installed `git-sizer`:

```bash
# Install git-sizer to your PATH
go install github.com/github/git-sizer@latest

# Build without embedded binaries (will use system git-sizer)
go build -tags noembedded -o bin/server cmd/server/main.go
```

## Updating git-sizer Version

To update the git-sizer version:

1. Set the `GIT_SIZER_VERSION` environment variable:
   ```bash
   GIT_SIZER_VERSION=1.5.0 ./scripts/download-git-sizer.sh
   ```

2. Rebuild the application:
   ```bash
   make build
   ```

## Binary Verification

The package automatically verifies extracted binaries by:
1. Checking file permissions (executable bit)
2. Running `--version` to ensure the binary is functional
3. Caching the path for subsequent uses

## Cleanup

Extracted binaries are stored in:
- Standard environments: `$TMPDIR/github-migrator-binaries/`
- Azure App Service: `/home/site/tmp/github-migrator-binaries/`
- Custom: `$GHMIG_TEMP_DIR/github-migrator-binaries/`

To clean up:

```go
import "github.com/kuhlman-labs/github-migrator/internal/embedded"

// Call during application shutdown
embedded.CleanupExtractedBinaries()
```

## Azure App Service Deployment

When deploying to Azure App Service, the application automatically detects the Azure environment (via `WEBSITE_SITE_NAME` environment variable) and uses `/home/site/tmp` for binary extraction instead of `/tmp`, which may have permission restrictions in Azure containers.

No additional configuration is required for Azure deployments.

## Troubleshooting

### Build Error: "pattern bin/git-sizer-*: no matching files found"

Run the download script before building:
```bash
./scripts/download-git-sizer.sh
```

### Runtime Error: "git-sizer binary not embedded"

The binary for your platform wasn't included in the build. Ensure all binaries are downloaded before building, or use the system git-sizer as a fallback.

### Runtime Error: "failed to get git-sizer binary"

Check that:
1. The binary was properly embedded during build
2. The application has permission to write to the temp directory
3. The extracted binary has execute permissions

**Azure App Service Specific**: If you see errors like "fork/exec /tmp/github-migrator-binaries/git-sizer: no such file or directory" in Azure App Service, the application automatically uses `/home/site/tmp` instead of `/tmp` when it detects the `WEBSITE_SITE_NAME` environment variable. This should resolve automatically, but you can also set `GHMIG_TEMP_DIR` to a custom path if needed.

## Architecture

```
internal/embedded/
├── binaries.go          # Embedding and extraction logic
├── bin/                 # Downloaded binaries (gitignored)
│   ├── git-sizer-linux-amd64
│   ├── git-sizer-linux-arm64
│   ├── git-sizer-darwin-amd64
│   ├── git-sizer-darwin-arm64
│   └── git-sizer-windows-amd64.exe
└── README.md            # This file

scripts/
└── download-git-sizer.sh  # Download script
```

## Security Considerations

1. **Binary Verification**: Binaries are downloaded from official GitHub releases
2. **Checksum Verification**: Consider adding SHA256 checksum verification
3. **Temporary Storage**: Extracted binaries are stored in the system temp directory with restrictive permissions
4. **Execution**: Only binaries embedded at build time can be executed

## Future Enhancements

- [ ] Add SHA256 checksum verification for downloaded binaries
- [ ] Support for additional architectures (ppc64le, s390x)
- [ ] Compressed binary storage to reduce binary size
- [ ] Automatic version detection and updates

