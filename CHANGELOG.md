# [1.3.0](https://github.com/dmakeienko/tfenvgo/compare/v1.2.0...v1.3.0) (2025-02-17)


### Bug Fixes

* "tfenvgo use" didnt set up correct permissions for the selected terraform version ([937f026](https://github.com/dmakeienko/tfenvgo/commit/937f02650da2ace974b4a12f412a983d352c204d))
* goconst ([07ab704](https://github.com/dmakeienko/tfenvgo/commit/07ab7043e82d2c72b7776b76745977eeaa8672cc))


### Features

* "tfenvgo install" now supports `min-required` and `latest-allowed` ([7cd421e](https://github.com/dmakeienko/tfenvgo/commit/7cd421e1de74b5fdcb7fb45f6d3ace300be879ab))
* "tfenvgo use" now supports `min-required` and `latest-allowed` ([1bf7713](https://github.com/dmakeienko/tfenvgo/commit/1bf7713b25079219e1212e3ddb121bde82899549))
* add arg validation to the `use`, `install`, `uninstall`; cosmetic changes in output ([3d9944b](https://github.com/dmakeienko/tfenvgo/commit/3d9944b9656a6b53a3e20709a8f49fe297819277))
* implement support for `.terraform-version` file ([74e93c1](https://github.com/dmakeienko/tfenvgo/commit/74e93c1a32638e567bf2ca4e9f09307e7586ab37))

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
