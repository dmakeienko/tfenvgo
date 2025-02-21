# tfenvgo

Terraform version manager like [tfenv](https://github.com/tfutils/tfenv) but written in Go.

> WARNING: this is my first project written in Go to learn it. It is inspired by `tfenv`, but is written without inspecting source, only documentation is used.

## Support

Currently `tfenvgo` supports the following OSes

* Linux
  * AMD64
  * ARM64
* macOS
  * AMD64
  * ARM64
* Windows - *Not supported and will not be*

## Installation

TBD

## Usage

### tfenvgo install [version]

Install a specific version of Terraform.
If no parameter is passed, the version to install is resolved automatically via **TFENVGO_TERRAFORM_VERSION** environment variable or **.terraform-version file**, in that order of precedence. If no argument provided, it will be defaulted to the `latest`.

**Available options:**

* `x.y.z` Semver 2.0.0 string specifying the exact version to install
* `latest` is a syntax to install latest available *stable* version
* (**TBD**) `latest:<regex>` is a syntax to install latest version matching regex
* `latest-allowed` is a syntax to scan your Terraform files to detect which version is maximally allowed
* `min-required` is a syntax to scan your Terraform files to detect which version is minimally required

**Available flags:**

* `--include-prerelease` - include prerelease versions into account when specifying `latest`, i.e. *1.12.0-alpha20250213*, *0.12.0-rc1* etc.

**Environment variables:**

`TFENVGO_ARCH` - specify to install binary for different architecture then your own.

`TFENVGO_OS_TYPE` - specify to install binary for different os_type then your own.

### tfenvgo use [version]

Switch a version to use.
If no parameter is passed, the version to use is resolved automatically via **TFENVGO_TERRAFORM_VERSION** environment variable or **.terraform-version file**, in that order of precedence, defaulting to `latest` if none are found.

**Available options:**

* `x.y.z` Semver 2.0.0 string specifying the exact version to use
* `latest` is a syntax to use latest installed *stable* version
* (**TBD**) `latest:<regex>` is a syntax to use latest version matching regex
* `min-required` is a syntax to scan your Terraform files to detect which version is minimally required
* `latest-allowed` is a syntax to scan your Terraform files to detect which version is latest allowed

### tfenvgo uninstall [version]

Uninstall a specific version of Terraform.

**Available options:**

* `x.y.z` Semver 2.0.0 string specifying the exact version to uninstall
* `latest` is a syntax to uninstall latest present version
* (**TBD**) `latest:<regex>` is a syntax to uninstall latest version matching regex

**Available flags:**

* `--include-prerelease` - include prerelease versions into account when specifying `latest`, i.e. *1.12.0-alpha20250213*, *0.12.0-rc1* etc.

### tfenvgo list

List all available terraform versions installed locally.
By default, it fetches *only stable* versions

**Available flags:**

* `--include-prerelease` - include prerelease versions i.e. *1.12.0-alpha20250213*, *0.12.0-rc1* etc.

### tfenvgo list-remote

Get all available versions of Terraform from the Hashicorp release page.
By default, it fetches *only stable* versions

**Available flags:**

* `--include-prerelease` - include prerelease versions i.e. *1.12.0-alpha20250213*, *0.12.0-rc1* etc.

### tfenvgo pin

Write current terraform version set by `tfenvgo` to the `.terraform-version` file.

### tfenvgo version (version-name)

Display current terraform version set by `tfenvgo`

## Environment variables

`TFENVGO_ARCH`

Specifies architecture. Default architecture is defined during compilation. Override to download terraform binary for other architecture.

`TFENVGO_OS_TYPE`

Specifies OS type. Default OS type is defined during compilation. Override to download terraform binary for OS.

`TFENVGO_TERRAFORM_VERSION`

If not empty string, this variable overrides Terraform version provided by `.terraform-version` file and commands `tfenvgo install`, `tfenvgo use`.

## .terraform-version file

If you put a `.terraform-version` file in your project root, `tfenvgo` detects it and uses the version written in it. If the version is latest or latest:<regex> (TBD), the latest matching version currently installed will be selected.

> NOTE: `TFENVGO_TERRAFORM_VERSION` environment variable can be used to override version, specified by `.terraform-version` file.

For `tfenvgo` to be able to detect `.terraform-version` file, add provided shell hook to your shell config (`.zshrc` or `.bashrc`)

```sh
cd() {
  builtin cd "$@" || return

  if [ -f ".terraform-version" ]; then
    tfenvgo use
  fi
}
```
