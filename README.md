# tfenvgo

Terraform version manager [tfenv](https://github.com/tfutils/tfenv) written in Go.

> WARNING: this is my first project written in Go to learn it. It is inpired by tfenv, but is written without inspecting source, only documentation is used.

## Support

Currently tfenv supports the following OSes

* Linux
  * AMD64
  * ARM - *TBD*
* macOS *TBD*
* Windows - *Not supported and will not be*

## Installation

TBD

## Usage

### tfenvgo install [version]

Install a specific version of Terraform.
If no parameter is passed, the version to install is resolved automatically via **TFENVGO_TERRAFORM_VERSION** environment variable or **.terraform-version file (TBD)**, in that order of precedence. If no argument provided, it will be defaulted to the `latest`.

**Available options:**

* `x.y.z` Semver 2.0.0 string specifying the exact version to install
* `latest` is a syntax to install latest available *stable* version
* (**TBD**) `latest:<regex>` is a syntax to install latest version matching regex
* (**TBD**) `latest-allowed` is a syntax to scan your Terraform files to detect which version is maximally allowed
* (**TBD**) `min-required` is a syntax to scan your Terraform files to detect which version is minimally required

**Environment variables:**

`TFENVGO_ARCH` - specify to install binary for different architecture then your own.

`TFENVGO_OS_TYPE` - specify to install binary for different os_type then your own.

### tfenvgo use [version]

Switch a version to use.
If no parameter is passed, the version to use is resolved automatically via **TFENVGO_TERRAFORM_VERSION** environment variable or **.terraform-version file (TBD)**, in that order of precedence, defaulting to `latest` if none are found.

**Available options:**

* `x.y.z` Semver 2.0.0 string specifying the exact version to use
* `latest` is a syntax to use latest installed *stable* version
* (**TBD**) `latest:<regex>` is a syntax to use latest version matching regex
* (**TBD**) `min-required` is a syntax to scan your Terraform files to detect which version is minimally required

### tfenv uninstall [version]

Uninstall a specific version of Terraform.

**Available options:**

* `x.y.z` Semver 2.0.0 string specifying the exact version to uninstall
* `latest` is a syntax to uninstall latest present version
* (**TBD**) `latest:<regex>` is a syntax to uninstall latest version matching regex
