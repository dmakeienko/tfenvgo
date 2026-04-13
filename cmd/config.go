package cmd

import (
	"os"
	"path/filepath"
	"runtime"
)

const terraformReleasesURL = "https://releases.hashicorp.com/terraform"

// getUserHomeDir safely gets the user home directory
func getUserHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
}

// Initialize paths safely
func initPaths() error {
	homeDir, err := getUserHomeDir()
	if err != nil {
		return err
	}

	rootURL = filepath.Join(homeDir, ".tfenvgo")
	terraformBinPath = filepath.Join(rootURL, "bin")
	terraformVersionPath = filepath.Join(rootURL, "versions")
	currentTerraformVersionPath = filepath.Join(terraformBinPath, "terraform")

	return nil
}

var (
	rootURL                     string
	terraformBinPath            string
	terraformVersionPath        string
	currentTerraformVersionPath string
)

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
