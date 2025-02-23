# tfenvgo

Terraform version manager like [tfenv](https://github.com/tfutils/tfenv) but written in Go.

> **WARNING:** This is my first project written in Go to learn it. It is inspired by `tfenv`, but is written without inspecting the source, only documentation is used.

## Why use `tfenvgo` instead of `tfenv`?

Here are the reasons:

1. Distributed as a single binary, easy to install.
2. `min-required` and `latest-allowed` don't have limitations as `tfenv`: `tfenvgo` uses real Go regex.
3. Ability to work with *pre-release* Terraform versions (see `--include-prerelease` flag).
4. `tfenvgo` is faster by around 50%.

## Support

Currently, `tfenvgo` supports the following OS:

* Linux
  * AMD64
  * ARM64
* macOS
  * AMD64
  * ARM64
* Windows - *Not supported and will not be*

## Installation

### Manual

1. Get the latest release:

    ```sh
    VERSION=$(curl -s "https://api.github.com/repos/dmakeienko/tfenvgo/releases" | jq -r '.[].tag_name' | head -1)
    ```

2. Download the archive:

    > **NOTE:** Don't forget to change `arch` and `os` if yours are different.

    ```sh
    curl -LO https://github.com/dmakeienko/tfenvgo/releases/download/$VERSION/tfenvgo-$VERSION-linux-amd64.tar.gz
    ```

3. Unarchive it:

    ```sh
    tar -xvzf tfenvgo-$VERSION-linux-amd64.tar.gz
    ```

4. Install `tfenvgo` into any location that is in your `PATH`:

    ```sh
    sudo mv tfenvgo /usr/local/bin
    ```

5. Update your shell profile:

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

Install a specific version of Terraform. If no parameter is passed, the version to install is resolved automatically via the **TFENVGO_TERRAFORM_VERSION** environment variable or **.terraform-version** file, in that order of precedence. If no argument is provided, it will default to the `latest`.

**Available options:**

* `x.y.z` - Semver 2.0.0 string specifying the exact version to install.
* `latest` - Syntax to install the latest available *stable* version.
* (**TBD**) `latest:<regex>` - Syntax to install the latest version matching the regex.
* `latest-allowed` - Syntax to scan your Terraform files to detect which version is maximally allowed.
* `min-required` - Syntax to scan your Terraform files to detect which version is minimally required.

**Available flags:**

* `--include-prerelease` - Include prerelease versions when specifying `latest`, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

**Environment variables:**

* `TFENVGO_ARCH` - Specify to install the binary for a different architecture than your own.
* `TFENVGO_OS_TYPE` - Specify to install the binary for a different OS type than your own.

### tfenvgo use [version]

Switch to a specific version to use. If no parameter is passed, the version to use is resolved automatically via the **TFENVGO_TERRAFORM_VERSION** environment variable or **.terraform-version** file, in that order of precedence, defaulting to `latest` if none are found.

**Available options:**

* `x.y.z` - Semver 2.0.0 string specifying the exact version to use.
* `latest` - Syntax to use the latest installed *stable* version.
* (**TBD**) `latest:<regex>` - Syntax to use the latest version matching the regex.
* `min-required` - Syntax to scan your Terraform files to detect which version is minimally required.
* `latest-allowed` - Syntax to scan your Terraform files to detect which version is the latest allowed.

### tfenvgo uninstall [version]

Uninstall a specific version of Terraform.

**Available options:**

* `x.y.z` - Semver 2.0.0 string specifying the exact version to uninstall.
* `latest` - Syntax to uninstall the latest present version.
* (**TBD**) `latest:<regex>` - Syntax to uninstall the latest version matching the regex.

**Available flags:**

* `--include-prerelease` - Include prerelease versions when specifying `latest`, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

### tfenvgo list

List all available Terraform versions installed locally. By default, it fetches *only stable* versions.

**Available flags:**

* `--include-prerelease` - Include prerelease versions, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

### tfenvgo list-remote

Get all available versions of Terraform from the Hashicorp release page. By default, it fetches *only stable* versions.

**Available flags:**

* `--include-prerelease` - Include prerelease versions, e.g., *1.12.0-alpha20250213*, *0.12.0-rc1*, etc.

### tfenvgo pin

Write the current Terraform version set by `tfenvgo` to the `.terraform-version` file.

### tfenvgo version (version-name)

Display the current Terraform version set by `tfenvgo`.

## Environment variables

* `TFENVGO_ARCH` - Specifies the architecture. The default architecture is defined during compilation. Override to download the Terraform binary for another architecture.
* `TFENVGO_OS_TYPE` - Specifies the OS type. The default OS type is defined during compilation. Override to download the Terraform binary for another OS.
* `TFENVGO_TERRAFORM_VERSION` - If not an empty string, this variable overrides the Terraform version provided by the `.terraform-version` file and commands `tfenvgo install`, `tfenvgo use`.

## .terraform-version file

If you put a `.terraform-version` file in your project root, `tfenvgo` detects it and uses the version written in it. If the version is `latest` or `latest:<regex>` (TBD), the latest matching version currently installed will be selected.

> **NOTE:** The `TFENVGO_TERRAFORM_VERSION` environment variable can be used to override the version specified by the `.terraform-version` file.

For `tfenvgo` to be able to detect the `.terraform-version` file, add the provided shell hook to your shell config (`.zshrc` or `.bashrc`):

```sh
cd() {
  builtin cd "$@" || return

  if [ -f ".terraform-version" ]; then
    tfenvgo use
  fi
}
```
