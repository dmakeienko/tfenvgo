# [1.2.0](https://github.com/dmakeienko/tfenvgo/compare/v1.1.0...v1.2.0) (2025-02-08)


### Features

* `install` command now supports `TFENVGO_ARCH` and `TFENVGO_OS_TYPE` env variables that allow to download Terraform binary for a different os/arch ([4d87bbc](https://github.com/dmakeienko/tfenvgo/commit/4d87bbcbd4838b4bc89c00e3ea28c96d80e76224))

# [1.1.0](https://github.com/dmakeienko/tfenvgo/compare/v1.0.1...v1.1.0) (2025-02-08)


### Bug Fixes

* golangci-lint: move "latest" to a constant ([43efd2f](https://github.com/dmakeienko/tfenvgo/commit/43efd2f6f1f5cd4a7d2bdb87d6d75dc40e66c22f))


### Features

* `list` now sorts versions from latest to oldest; `install` now accepts `latest` as an argument ([0f57d2b](https://github.com/dmakeienko/tfenvgo/commit/0f57d2b8111c8760f4364e4f6d375d9e10a02f43))
* `tfenvgo install` now supports version from TFENVGO_TERRAFORM_VERSION env variable; if no argument provided, argument default to `latest` ([8ba244d](https://github.com/dmakeienko/tfenvgo/commit/8ba244deab63c18b3bd4fe302cb1458e385b40ba))
* `tfenvgo uninstall` can now accept `latest` as an argument ([5b1bae3](https://github.com/dmakeienko/tfenvgo/commit/5b1bae3c8f275839421858a93799daead2c0334b))
* `tfenvgo use` can now accept `latest` as an argument ([324f899](https://github.com/dmakeienko/tfenvgo/commit/324f899a0bad3bfee1ff6477139fdcf0eff0e57a))
* add `list-remote` command to fetch all available stable terraform versions ([c81ff98](https://github.com/dmakeienko/tfenvgo/commit/c81ff98c792ac5081710c722513c80435bfbdcde))
* support `latest` version in `install` command ([6df5df9](https://github.com/dmakeienko/tfenvgo/commit/6df5df9c1ac323b09279196db3f032e74ed5a6d7))

## [1.0.1](https://github.com/dmakeienko/tfenvgo/compare/v1.0.0...v1.0.1) (2025-02-05)


### Bug Fixes

* G110: Potential DoS vulnerability via decompression bomb ([170460a](https://github.com/dmakeienko/tfenvgo/commit/170460ac159db3d7c6e64ef89f401c0dd88fbfe7))
* lint errcheck, stylecheck, other ([df64edd](https://github.com/dmakeienko/tfenvgo/commit/df64edd1b8cbc599f7b06306589f40e620ea252f))
