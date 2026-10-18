# Installation Guide

`tf-move-gen` is distributed as a standalone, zero-dependency static binary for macOS, Linux, and Windows. It requires no runtime, no Python, no Node.js, and no C libraries.

---

## Installation Methods

### 1. Homebrew (macOS & Linux) — Recommended

If you use Homebrew, install `tf-move-gen` from the official tap:

```bash
brew install smford/tap/tf-move-gen
```

Or tap first and then install:

```bash
brew tap smford/homebrew-tap
brew install tf-move-gen
```

To update in the future:

```bash
brew update
brew upgrade tf-move-gen
```

---

### 2. Precompiled Binaries (macOS, Linux, Windows)

Precompiled static binaries are generated for each release. You can download and unpack them directly.

#### Automated Install Script (macOS & Linux)

Install the latest binary to `/usr/local/bin` using `curl`:

```bash
curl -fsSL https://raw.githubusercontent.com/smford/tf-move-gen/main/scripts/install.sh | bash
```

Or install manually:

#### macOS (Apple Silicon M1/M2/M3/M4)

```bash
curl -sSL -O https://github.com/smford/tf-move-gen/releases/latest/download/tf-move-gen_0.1.0_darwin_arm64.tar.gz
tar -xzf tf-move-gen_0.1.0_darwin_arm64.tar.gz
sudo mv tf-move-gen /usr/local/bin/
rm tf-move-gen_0.1.0_darwin_arm64.tar.gz
```

#### macOS (Intel)

```bash
curl -sSL -O https://github.com/smford/tf-move-gen/releases/latest/download/tf-move-gen_0.1.0_darwin_amd64.tar.gz
tar -xzf tf-move-gen_0.1.0_darwin_amd64.tar.gz
sudo mv tf-move-gen /usr/local/bin/
rm tf-move-gen_0.1.0_darwin_amd64.tar.gz
```

#### Linux (x86_64 / amd64)

```bash
curl -sSL -O https://github.com/smford/tf-move-gen/releases/latest/download/tf-move-gen_0.1.0_linux_amd64.tar.gz
tar -xzf tf-move-gen_0.1.0_linux_amd64.tar.gz
sudo mv tf-move-gen /usr/local/bin/
rm tf-move-gen_0.1.0_linux_amd64.tar.gz
```

#### Linux (ARM64 / aarch64)

```bash
curl -sSL -O https://github.com/smford/tf-move-gen/releases/latest/download/tf-move-gen_0.1.0_linux_arm64.tar.gz
tar -xzf tf-move-gen_0.1.0_linux_arm64.tar.gz
sudo mv tf-move-gen /usr/local/bin/
rm tf-move-gen_0.1.0_linux_arm64.tar.gz
```

#### Windows

1. Download [`tf-move-gen_0.1.0_windows_amd64.zip`](https://github.com/smford/tf-move-gen/releases/latest/download/tf-move-gen_0.1.0_windows_amd64.zip).
2. Extract the archive.
3. Move `tf-move-gen.exe` into a folder included in your system `%PATH%` (e.g. `C:\Windows\System32` or your custom CLI tools directory).

---

### 3. Go Install (Go 1.23+)

If you already have Go installed, compile and install directly into `$GOPATH/bin`:

```bash
go install github.com/smford/tf-move-gen/cmd/tf-move-gen@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is in your `$PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

---

### 4. Docker Container Image

You can run `tf-move-gen` inside a minimal, scratch-based distroless container without installing any local binaries:

```bash
# Pull the latest image
docker pull ghcr.io/smford/tf-move-gen:latest

# Pipe plan directly into Docker
terraform show -json tfplan | docker run --rm -i ghcr.io/smford/tf-move-gen:latest --explain
```

---

### 5. Build from Source

To compile the latest development version:

```bash
# Clone the repository
git clone https://github.com/smford/tf-move-gen.git
cd tf-move-gen

# Build static binary using Makefile
make build

# The binary is located at bin/tf-move-gen
./bin/tf-move-gen --version

# Optionally install to your $GOPATH/bin
make install
```

---

## Verifying Your Installation

Confirm that `tf-move-gen` is properly installed and accessible:

```bash
tf-move-gen --version
```

Expected output:
```text
tf-move-gen v0.1.0 (commit: ..., built at: ...)
```

Display CLI usage options:

```bash
tf-move-gen --help
```

---

## Next Steps

- Follow the [Getting Started Tutorial](getting-started.md) to walk through a real-world refactor.
- Review the [CLI Reference](cli-reference.md) for all available flags and options.
- Check out [CI/CD Integration](ci-cd-integration.md) to add safety gates to your PR workflows.
