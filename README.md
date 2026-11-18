# tfenvgo

Terraform version manager like [tfenv](https://github.com/tfutils/tfenv) but written in Go. Now supports Opentofu!

> **WARNING:** This is my first project written in Go to learn it. It is inspired by `tfenv`, but is written without inspecting the source, only documentation is used.

## OpenTofu support

`tfenvgo` manages both **Terraform** and **OpenTofu** (`tofu`) side-by-side. Use the `--flavor` flag or the `flavor` command to switch between them.

## Why use `tfenvgo` instead of `tfenv`?

Here are the reasons:

1. Distributed as a single binary, easy to install.
2. `latest "regex"` uses real Go regular expressions, and `min-required` / `latest-allowed` use full SemVer constraint evaluation.
3. Ability to work with *pre-release* Terraform versions (see `--include-prerelease` flag).
4. `tfenvgo` is faster by around 50%.
5. Supports opentofu out of the box
6. Verifies the SHA-256 checksum of every downloaded release (see [Download verification](#download-verification)).

## Support

Currently, `tfenvgo` supports the following OS and archs:

* Linux
  * AMD64
  * ARM64
* macOS
  * AMD64
  * ARM64
* Windows - *Not supported and will not be*

### Quickstart

```sh
# Set OpenTofu as the persistent default
tfenvgo flavor tofu

# Install and activate the latest OpenTofu release
tfenvgo install latest
tfenvgo use latest

# Now ~/.tfenvgo/bin/tofu points to the installed binary
tofu version

# Override flavor for a single command without changing the default
tfenvgo install latest --flavor terraform
tfenvgo list --flavor terraform
```

Both flavors are fully isolated — installed versions live under separate directories and have independent active-version symlinks:

```text
~/.tfenvgo/
├── flavor                          # persisted default: "tofu" or "terraform"
├── bin/
│   ├── terraform -> ../versions/terraform/1.5.7/terraform
│   └── tofu      -> ../versions/tofu/1.8.0/tofu
└── versions/
    ├── terraform/1.5.7/terraform
    └── tofu/1.8.0/tofu
```

### Flavor resolution order

1. `--flavor <value>` flag (highest priority)
2. `TFENVGO_FLAVOR` environment variable
3. `~/.tfenvgo/flavor` file (written by `tfenvgo flavor <value>`)
4. `terraform` (built-in default)

### tfenvgo flavor [tofu|terraform]

Get or set the default binary flavor.

```sh
# Print the currently resolved flavor and where it comes from
tfenvgo flavor

# Persist "tofu" as the default for all future invocations
tfenvgo flavor tofu

# Switch back to terraform
tfenvgo flavor terraform
```

## Installation

### Homebrew

```sh
brew tap dmakeienko/tap
brew install tfenvgo
```

To upgrade to the latest version:

```sh
brew upgrade tfenvgo
```

### Go

Requires the Go version listed in `go.mod`:

```sh
go install github.com/dmakeienko/tfenvgo/v2@latest
```

### Manual

1. Download a release, verify its checksum and unpack it:

  ```sh
  # Latest release version (without the leading "v"), or set it yourself, e.g. VERSION=2.0.3
  VERSION=$(curl -fsSL https://api.github.com/repos/dmakeienko/tfenvgo/releases/latest | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p')
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m); case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; esac
  FILE="tfenvgo_${VERSION}_${OS}_${ARCH}.tar.gz"
  BASE="https://github.com/dmakeienko/tfenvgo/releases/download/v${VERSION}"

  curl -fsSLO "$BASE/$FILE"
  curl -fsSLO "$BASE/checksums.txt"
  grep " $FILE\$" checksums.txt | sha256sum -c -   # on macOS: shasum -a 256 -c -
  tar -xzf "$FILE" tfenvgo
  ```

2. Install `tfenvgo` into any location that is in your `PATH`:

  ```sh
  sudo mv tfenvgo /usr/local/bin
  ```

3. Update your shell profile:

  Add the following line to your shell config file:

  ```sh
  export PATH=$PATH:$HOME/.tfenvgo/bin
  ```

  Optionally, run:

  ```sh
  tfenvgo init
  ```

  This command will precreate the `$HOME/.tfenvgo/bin` folder structure.

## Usage

### tfenvgo install [version]

Install a specific version of Terraform or OpenTofu, verifying its checksum (see [Download verification](#download-verification)). Installing does not change the active version; use `tfenvgo use` for that. If no version is passed, it is taken from the **TFENVGO_VERSION** environment variable, then **TFENVGO_TERRAFORM_VERSION**, then the **.terraform-version** file, and defaults to `latest`.

**Available options:**

* `x.y.z` - Semver 2.0.0 string specifying the exact version to install.
* `latest` - Syntax to install the latest available *stable* version.
* `latest-allowed` - Syntax to scan your Terraform files to detect which version is maximally allowed.
* `min-required` - Syntax to scan your Terraform files to detect which version is minimally required.
* `latest "regex"` - Syntax to install the latest version matching the regex, e.g. `tfenvgo install latest "^1\.5\."`.

> NOTE: because some symbols interpreted by shell as commands, use quotes (" or ') to specify regex.
> NOTE: the regex is matched against versions like `1.5.7` or `1.12.0-alpha20250213`. Add `--include-prerelease` to consider pre-release versions too.

**Available flags:**

* `--include-prerelease` - Include prerelease versions when specifying `latest`, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

**Environment variables:**

* `TFENVGO_ARCH` - Specify to install the binary for a different architecture than your own.
* `TFENVGO_OS_TYPE` - Specify to install the binary for a different OS type than your own.
* `TFENVGO_DOWNLOAD_TIMEOUT` - Timeout for downloading the binary. Defaults to `300` seconds.

### tfenvgo use [version]

Switch to a specific version, installing it first if needed. If no version is passed, it is resolved the same way as for `tfenvgo install`: **TFENVGO_VERSION**, then **TFENVGO_TERRAFORM_VERSION**, then **.terraform-version**, then `latest`. Keywords such as `latest` are resolved against the upstream releases, not only the installed versions.

**Available options:**

* `x.y.z` - Semver 2.0.0 string specifying the exact version to use.
* `latest` - Syntax to use the latest available *stable* version (installing it if needed).
* `min-required` - Syntax to scan your Terraform files to detect which version is minimally required.
* `latest-allowed` - Syntax to scan your Terraform files to detect which version is the latest allowed.
* `latest "regex"` - Syntax to use the latest version matching the regex.

> NOTE: because some symbols interpreted by shell as commands, use quotes (" or ') to specify regex.
> NOTE: the regex is matched against versions like `1.5.7` or `1.12.0-alpha20250213`. Add `--include-prerelease` to consider pre-release versions too.

### tfenvgo uninstall version

Uninstall a specific version of Terraform or OpenTofu. A version is required (or set via **TFENVGO_VERSION** / **TFENVGO_TERRAFORM_VERSION**); running `tfenvgo uninstall` with nothing set is an error rather than removing a version for you. If you uninstall the active version, its `~/.tfenvgo/bin` symlink is removed too.

**Available options:**

* `x.y.z` - Semver 2.0.0 string specifying the exact version to uninstall.
* `latest` - Syntax to uninstall the latest present version.
* `latest "regex"` - Syntax to uninstall the latest installed version matching the regex.

> NOTE: because some symbols interpreted by shell as commands, use quotes (" or ') to specify regex.
> NOTE: the regex is matched against versions like `1.5.7` or `1.12.0-alpha20250213`. Add `--include-prerelease` to consider pre-release versions too.

**Available flags:**

* `--include-prerelease` - Include prerelease versions when specifying `latest`, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

### tfenvgo list

List all available Terraform versions installed locally. By default, it fetches *only stable* versions.

**Available flags:**

* `--include-prerelease` - Include prerelease versions, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

### tfenvgo list-remote

Get all available versions from the HashiCorp release page (Terraform) or the OpenTofu GitHub releases (`--flavor tofu`), newest first, one per line, so the output can be piped (`tfenvgo list-remote | head -5`). By default, it fetches *only stable* versions.

**Available flags:**

* `--include-prerelease` - Include prerelease versions, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

### tfenvgo pin

Write the current Terraform version set by `tfenvgo` to the `.terraform-version` file.

### tfenvgo version (version-name)

Display the current Terraform or OpenTofu version set by `tfenvgo`. To print the version of `tfenvgo` itself, use `tfenvgo --version`.

### Shell completion

`tfenvgo completion bash|zsh|fish|powershell` prints a completion script. `use` and `uninstall` complete installed versions; `flavor` completes `terraform` and `tofu`.

## Environment variables

* `TFENVGO_VERSION` - Version used by `tfenvgo install` and `tfenvgo use` when no version argument is passed. Takes precedence over `TFENVGO_TERRAFORM_VERSION` and the `.terraform-version` file.
* `TFENVGO_TERRAFORM_VERSION` - Same as `TFENVGO_VERSION`, kept for compatibility; used only when `TFENVGO_VERSION` is not set.
* `TFENVGO_FLAVOR` - `terraform` or `tofu`. Overrides `~/.tfenvgo/flavor`; the `--flavor` flag overrides it. Any other value is an error.
* `TFENVGO_ARCH` - Specifies the architecture (e.g. `amd64`, `arm64`). The default architecture is defined during compilation. Override to download the binary for another architecture.
* `TFENVGO_OS_TYPE` - Specifies the OS type (e.g. `linux`, `darwin`). The default OS type is defined during compilation. Override to download the binary for another OS.
* `NO_COLOR` - Set to any non-empty value to disable colored output. Colors are also disabled automatically when output is not a terminal.
* `TFENVGO_DOWNLOAD_TIMEOUT` - Timeout for downloading a binary, applied to both the HTTP client and the request context. Accepts a plain number of seconds (`600`) or a Go duration string (`10m`, `90s`). Defaults to `300` seconds. Raise it for large binaries or slow/unstable connections.

## Download verification

Before extracting a release, `tfenvgo` downloads the release's `SHA256SUMS` file from the same release location (`terraform_<version>_SHA256SUMS` from releases.hashicorp.com, `tofu_<version>_SHA256SUMS` from the OpenTofu GitHub release) and checks the archive's SHA-256 against it. The install fails if the checksum file is missing, does not list the archive, or the checksum does not match.

> NOTE: this protects against corrupted and truncated downloads. The checksum file comes from the same server as the archive, so it does not prove authenticity on its own; GPG signature verification is not implemented yet.

## Exit codes

Every command exits with status `1` on failure, so `tfenvgo install 1.5.7 && terraform plan` stops if the install fails.

## .terraform-version file

If you put a `.terraform-version` file in the current directory, `tfenvgo install` and `tfenvgo use` without a version argument use the version written in it. The file must contain an exact version such as `1.5.7` (an optional leading `v` is allowed); the first such line is used. `tfenvgo pin` writes this file for you. Keywords such as `latest` are not read from the file; when it has no exact version, `tfenvgo` falls back to `latest`.

> **NOTE:** The `TFENVGO_VERSION` and `TFENVGO_TERRAFORM_VERSION` environment variables override the version specified by the `.terraform-version` file. Only the current directory is checked, not parent directories.

For `tfenvgo` to be able to detect the `.terraform-version` file, add the provided shell hook to your shell config (`.zshrc` or `.bashrc`):

```sh
cd() {
  builtin cd "$@" || return

  if [ -f ".terraform-version" ]; then
    tfenvgo use
  fi
}
```

## SemVer evaluation

`tfenvgo` uses [SemVer package](https://github.com/Masterminds/semver) to parse, sort and evaluate constraints.

### Hyphen Range Comparisons

There are multiple methods to handle ranges and the first is hyphens ranges. These look like:

```sh
1.2 - 1.4.5 which is equivalent to >= 1.2 <= 1.4.5
2.3.4 - 4.5 which is equivalent to >= 2.3.4 <= 4.5
```

>NOTE: 1.2-1.4.5 without whitespace is parsed completely differently; it's parsed as a single constraint 1.2.0 with prerelease 1.4.5.

### Wildcards In Comparisons

The x, X, and * characters can be used as a wildcard character. This works for all comparison operators. When used on the = operator it falls back to the patch level comparison (see tilde below). For example,

```sh
1.2.x is equivalent to >= 1.2.0, < 1.3.0
>= 1.2.x is equivalent to >= 1.2.0
<= 2.x is equivalent to < 3
* is equivalent to >= 0.0.0
```

### Tilde Range Comparisons (Patch)

The tilde (~) comparison operator is for patch level ranges when a minor version is specified and major level changes when the minor number is missing. For example,

```sh
~1.2.3 is equivalent to >= 1.2.3, < 1.3.0
~1 is equivalent to >= 1, < 2
~2.3 is equivalent to >= 2.3, < 2.4
~1.2.x is equivalent to >= 1.2.0, < 1.3.0
~1.x is equivalent to >= 1, < 2
```

### Caret Range Comparisons (Major)

The caret (^) comparison operator is for major level changes once a stable (1.0.0) release has occurred. Prior to a 1.0.0 release the minor versions acts as the API stability level. This is useful when comparisons of API versions as a major change is API breaking. For example,

```sh
^1.2.3 is equivalent to >= 1.2.3, < 2.0.0
^1.2.x is equivalent to >= 1.2.0, < 2.0.0
^2.3 is equivalent to >= 2.3, < 3
^2.x is equivalent to >= 2.0.0, < 3
^0.2.3 is equivalent to >=0.2.3 <0.3.0
^0.2 is equivalent to >=0.2.0 <0.3.0
^0.0.3 is equivalent to >=0.0.3 <0.0.4
^0.0 is equivalent to >=0.0.0 <0.1.0
^0 is equivalent to >=0.0.0 <1.0.0
```
