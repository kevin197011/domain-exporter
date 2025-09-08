# GitHub Actions Workflows

This directory contains GitHub Actions workflows for automated CI/CD pipeline.

## Workflows

### 1. CI Pipeline (`.github/workflows/ci.yml`)

Runs on every push and pull request to main/develop branches:

- **Code Quality Checks**:
  - Go formatting (`go fmt`)
  - Go vetting (`go vet`)
  - Linting with `golangci-lint`
  - Security scanning with `gosec`

- **Testing**:
  - Unit tests with race detection
  - Code coverage reporting to Codecov

- **Helm Chart Validation**:
  - Helm chart linting
  - Template validation with different values

### 2. Docker Build & Push (`.github/workflows/docker-build-push.yml`)

Builds and pushes Docker images to GitHub Container Registry:

- **Triggers**:
  - Push to `main` or `develop` branches
  - Git tags starting with `v*`
  - Pull requests (build only, no push)

- **Features**:
  - Multi-platform builds (linux/amd64, linux/arm64)
  - Automatic tagging based on branch/tag
  - Build cache optimization
  - Artifact attestation for security

- **Image Tags**:
  - `latest` for main branch
  - `develop` for develop branch
  - Semantic versioning for tags (e.g., `v1.2.3`, `v1.2`, `v1`)

### 3. Release Pipeline (`.github/workflows/release.yml`)

Automated release process triggered by version tags:

- **Binary Builds**:
  - Cross-platform compilation (Linux, macOS, Windows)
  - Multiple architectures (amd64, arm64)

- **Docker Images**:
  - Multi-platform Docker images
  - Semantic version tagging

- **GitHub Release**:
  - Automatic changelog generation
  - Binary attachments
  - Helm chart packaging
  - Release notes from CHANGELOG.md or git commits

- **Helm Chart Updates**:
  - Version bumping in Chart.yaml
  - Image tag updates in values files

### 4. Dependency Updates (`.github/dependabot.yml`)

Automated dependency management:

- **Go Modules**: Weekly updates for Go dependencies
- **GitHub Actions**: Weekly updates for action versions
- **Docker**: Weekly updates for base images

## Setup Instructions

### 1. Enable GitHub Container Registry

1. Go to your repository settings
2. Navigate to "Actions" → "General"
3. Under "Workflow permissions", select "Read and write permissions"
4. Check "Allow GitHub Actions to create and approve pull requests"

### 2. Configure Secrets (Optional)

For enhanced functionality, you may want to add these secrets:

- `CODECOV_TOKEN`: For code coverage reporting (optional, works without token for public repos)

### 3. Repository Settings

Ensure your repository has these settings:

- **Packages**: Enable "Inherit access from source repository"
- **Actions**: Enable "Allow all actions and reusable workflows"

## Usage Examples

### Triggering a Release

Create and push a version tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will:
1. Build multi-platform binaries
2. Create Docker images with version tags
3. Generate a GitHub release with binaries and Helm chart
4. Update Helm chart versions

### Using the Docker Image

Pull the latest image:

```bash
docker pull ghcr.io/kevin197011/domain-exporter:latest
```

Pull a specific version:

```bash
docker pull ghcr.io/kevin197011/domain-exporter:v1.0.0
```

### Installing with Helm

Using the packaged chart from releases:

```bash
# Download the chart from GitHub releases
wget https://github.com/kevin197011/domain-exporter/releases/download/v1.0.0/domain-exporter-1.0.0.tgz

# Install the chart
helm install domain-exporter domain-exporter-1.0.0.tgz
```

Or directly from the repository:

```bash
helm install domain-exporter ./helm/domain-exporter
```

## Image Registry

Images are published to GitHub Container Registry (ghcr.io):

- **Registry**: `ghcr.io/kevin197011/domain-exporter`
- **Public Access**: Images are publicly accessible
- **Multi-Architecture**: Supports both AMD64 and ARM64

## Branch Strategy

- **main**: Production-ready code, triggers `latest` tag
- **develop**: Development branch, triggers `develop` tag
- **feature/***: Feature branches, only runs CI tests
- **v***: Version tags, triggers full release pipeline

## Monitoring

You can monitor the workflows in the "Actions" tab of your GitHub repository:

- View build logs and test results
- Monitor deployment status
- Check security scan results
- Review dependency update PRs