# [1.7.0](https://github.com/dmakeienko/tfenvgo/compare/v1.6.1...v1.7.0) (2025-08-15)


### Bug Fixes

* G115: integer overflow conversion uint64 -> int64 (gosec) ([a75917d](https://github.com/dmakeienko/tfenvgo/commit/a75917d49337b77906dbc7b49410da825a3c0c1f))
* getTerraformVersionConstraint() didnt close file ([6c2788c](https://github.com/dmakeienko/tfenvgo/commit/6c2788c4c4dcc03706f7bd234fa87ee0ecf0b9e8))


### Features

* add logger ([b3d4639](https://github.com/dmakeienko/tfenvgo/commit/b3d4639e6b44733dfda294371c9da3d2b081eda6))
* zip bomb vulnerability handling; use HTTPS to for remote calls ([7e23a47](https://github.com/dmakeienko/tfenvgo/commit/7e23a47bb97a12ec79f0d750c4ee6cdd1331ddd5))

## [1.6.1](https://github.com/dmakeienko/tfenvgo/compare/v1.6.0...v1.6.1) (2025-02-24)


### Bug Fixes

* fix getTerraformVersionConstraint() looking all directories recursively and encointering error that prevented proper constraint find ([1591e51](https://github.com/dmakeienko/tfenvgo/commit/1591e512cba3f050fc71c79613bc96882c6d42b9))

# [1.6.0](https://github.com/dmakeienko/tfenvgo/compare/v1.5.1...v1.6.0) (2025-02-23)


### Features

* implement `install/use/uninstall` with regex ([f2483c8](https://github.com/dmakeienko/tfenvgo/commit/f2483c89ace8a9a8fbbd9ad14d4d3b5f032bde5b))

## [1.5.1](https://github.com/dmakeienko/tfenvgo/compare/v1.5.0...v1.5.1) (2025-02-22)


### Bug Fixes

* fix `install` command allowing to download nonexistent files ([e71e010](https://github.com/dmakeienko/tfenvgo/commit/e71e0106d4d4acc2388134de82af4e3950b393b8))
* remove shell update func to prevent conflicts; update docs; `tfenvgo init` is now optional step, folder structure will be created automatically when executing `tfenvgo use` ([c3c7699](https://github.com/dmakeienko/tfenvgo/commit/c3c7699817b2caeebd3508ccfcf60dbb7238e60a))

# [1.5.0](https://github.com/dmakeienko/tfenvgo/compare/v1.4.1...v1.5.0) (2025-02-21)


### Bug Fixes

* fix behaviour of the `use` command: now checks if the required version is installed and installs it if it is not present ([4b0d90d](https://github.com/dmakeienko/tfenvgo/commit/4b0d90d6e8f5b2c2361c59912766bd2a8f816bfe))


### Features

* `list`, `list-remote` can now fetch prerelease versions with `--include-prerelease` flag; commands `install`, `uninstall`, `use` can use flag `--include-prerelease` while using `latest` as an argument ([50209cb](https://github.com/dmakeienko/tfenvgo/commit/50209cbdbb9e5ddf35f4b53c58bbcdee041bef64))
* add `pin` and  `version-name` command ([c6940a1](https://github.com/dmakeienko/tfenvgo/commit/c6940a13908b4be266c77b7d4708b0507463bb7f))

## [1.4.1](https://github.com/dmakeienko/tfenvgo/compare/v1.4.0...v1.4.1) (2025-02-18)


### Bug Fixes

* fix `latest-allowed` and `min-required` commands using only remote versions ([2e74cb1](https://github.com/dmakeienko/tfenvgo/commit/2e74cb1d2cd46e891ce41aed787de20a79474644))

# [1.4.0](https://github.com/dmakeienko/tfenvgo/compare/v1.3.0...v1.4.0) (2025-02-17)


### Bug Fixes

* fix behaviour of `TFENVGO_TERRAFORM_VERSION` env var and `.terraform-version` file ([f6ed64e](https://github.com/dmakeienko/tfenvgo/commit/f6ed64e4e962bd858b098d54008c25320bef4f5c))


### Features

* show current selected version using `tfenvgo list` command ([d2d6033](https://github.com/dmakeienko/tfenvgo/commit/d2d60333efe50e7af04e53b4279862549543a71f))

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
