package cmd

import (
	"os"
	"path/filepath"
	"runtime"
)

const terraformReleasesURL = "https://releases.hashicorp.com/terraform"

var rootURL = filepath.Join(os.Getenv("HOME"), ".tfenvgo")
var terraformBinPath = filepath.Join(rootURL, "bin")
var terraformVersionPath = filepath.Join(rootURL, "versions")
var currentTerraformVersionPath = filepath.Join(terraformBinPath, "terraform")

// System
var defaultArch = runtime.GOARCH
var defaultOSType = runtime.GOOS

// Colors
var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

// Environment variables
const archEnvKey = "TFENVGO_ARCH"
const osTypeEnvKey = "TFENVGO_OS_TYPE"
const terraformVersionEnvKey = "TFENVGO_TERRAFORM_VERSION"

// Arguments
const (
	latestArg        = "latest"
	latestAllowedArg = "latest-allowed"
	minRequiredArg   = "min-required"
)

const terraformVersionFilename string = ".terraform-version"

// flags
var PreReleaseVersionsIncluded bool
