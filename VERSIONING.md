# Versioning with Git Tags

This project uses Git tags for version management, similar to Python's setuptools_scm approach.

## How it works

- Version numbers are based on Git tags (e.g., `v1.2.3`)
- Go modules automatically use Git tags as version numbers
- No manual version management needed in code
- Releases are triggered by creating and pushing Git tags

## Version Format

- **Semantic versioning**: `vMAJOR.MINOR.PATCH` (e.g., `v1.2.3`)
- **Pre-releases**: `v1.2.3-beta.1`, `v1.2.3-rc.1`
- **Go module compatible**: Tags must start with `v`

## Creating a Release

### 1. Development Phase
Developers work normally, committing changes to the repository.

### 2. Version Tagging
When ready to release, create and push a Git tag:

```bash
# Create a new version tag
git tag v1.0.0

# Push the tag to trigger release
git push origin v1.0.0
```

### 3. What happens automatically
- GitHub Actions detects the new tag
- Runs tests and builds the project
- Creates a GitHub release
- Go module proxy automatically indexes the new version

### 4. Users can install specific versions
```bash
# Install latest version
go get github.com/WaveSpeedAI/wavespeed-go

# Install specific version
go get github.com/WaveSpeedAI/wavespeed-go@v1.0.0

# Install pre-release
go get github.com/WaveSpeedAI/wavespeed-go@v1.1.0-beta.1
```

## Version Types

| Command | When to use | Example |
|---------|-------------|---------|
| `git tag v1.0.0` | First stable release | v1.0.0 |
| `git tag v1.1.0` | New features, backwards compatible | v1.1.0 |
| `git tag v2.0.0` | Breaking changes | v2.0.0 |
| `git tag v1.0.1` | Bug fixes | v1.0.1 |

## Release Workflow

```bash
# 1. Ensure all changes are committed
git add .
git commit -m "feat: add new feature"

# 2. Create version tag
git tag v1.0.0

# 3. Push both commits and tags
git push origin main
git push origin --tags

# 4. GitHub Actions automatically creates release
```

## Pre-release Versions

For beta/rc versions:

```bash
git tag v1.0.0-beta.1
git tag v1.0.0-rc.1
git push origin --tags
```

## Checking Current Version

```bash
# Check latest tag
git describe --tags --abbrev=0

# Check all tags
git tag --list

# Check Go module info
go list -m -versions github.com/WaveSpeedAI/wavespeed-go
```

## GitHub Actions Workflow

The release process is automated when tags are pushed:

```yaml
name: Release
on:
  push:
    tags: ['v*.*.*']
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
      - run: go test -v ./...
  release:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
      - run: go test -v ./...
      - run: go build ./...
      - run: Create GitHub release
```

## Best Practices

### When to Release
- **Patch releases** (`v1.0.1`): Bug fixes, security updates
- **Minor releases** (`v1.1.0`): New features, backwards compatible
- **Major releases** (`v2.0.0`): Breaking changes

### Tag Naming
- Always use `v` prefix for Go module compatibility
- Use semantic versioning
- Test tags locally before pushing

### Testing
- Run `go test -v ./...` before creating tags
- Consider using pre-release tags for testing: `v1.0.0-beta.1`

## Troubleshooting

### Common Issues

1. **Tag already exists**: Delete and recreate
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   git tag v1.0.0
   ```

2. **Go proxy delay**: New versions may take time to appear
   ```bash
   go clean -modcache
   go get github.com/WaveSpeedAI/wavespeed-go@latest
   ```

3. **GitHub Actions not triggered**: Ensure tag matches pattern `v*.*.*`

## Comparison with Python setuptools_scm

| Aspect | Go Git Tags | Python setuptools_scm |
|--------|-------------|----------------------|
| Version source | Git tag | Git tag + commit count |
| Version format | v1.2.3 | 1.2.3.dev4+g1234567 |
| Release trigger | Git tag push | Git tag push |
| Package registry | Go module proxy | PyPI |
| Installation | go get | pip install |

This approach provides the same manual control as Python's setuptools_scm while leveraging Go's native module versioning.
