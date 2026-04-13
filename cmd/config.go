package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	FlavorTerraform = "terraform"
	FlavorTofu      = "tofu"

	terraformReleasesURL = "https://releases.hashicorp.com/terraform"
	tofuReleasesAPI      = "https://api.github.com/repos/opentofu/opentofu/releases"
	tofuDownloadBase     = "https://github.com/opentofu/opentofu/releases/download"
)

// getUserHomeDir safely gets the user home directory
func getUserHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
}

// versionsDir returns the flavor-specific versions directory.
func versionsDir(flv string) string {
	return filepath.Join(rootURL, "versions", flv)
}

// binaryName returns the binary name for the given flavor.
func binaryName(flv string) string {
	if flv == FlavorTofu {
		return FlavorTofu
	}
	return FlavorTerraform
}

// currentSymlinkPath returns the path to the active-version symlink for the flavor.
func currentSymlinkPath(flv string) string {
	return filepath.Join(rootURL, "bin", binaryName(flv))
}

// installedBinaryPath returns the path to the installed binary for the given flavor+version.
func installedBinaryPath(flv, version string) string {
	return filepath.Join(versionsDir(flv), version, binaryName(flv))
}

// resolveFlavor returns the effective flavor for this invocation.
// Precedence: --flavor flag > TFENVGO_FLAVOR env > ~/.tfenvgo/flavor file > terraform default.
func resolveFlavor(flag string) string {
	if flag != "" {
		return flag
	}
	if env := os.Getenv(flavorEnvKey); env != "" {
		return env
	}
	flavorFile := filepath.Join(rootURL, "flavor")
	data, err := os.ReadFile(flavorFile) // #nosec G304 - path is constructed from controlled rootURL
	if err == nil {
		f := strings.TrimSpace(string(data))
		if f == FlavorTofu || f == FlavorTerraform {
			return f
		}
	}
	return FlavorTerraform
}

// initPaths initializes global path variables and runs the one-time migration
// of old-style ~/.tfenvgo/versions/<semver>/ layouts to the new
// ~/.tfenvgo/versions/terraform/<semver>/ layout.
func initPaths() error {
	homeDir, err := getUserHomeDir()
	if err != nil {
		return err
	}

	rootURL = filepath.Join(homeDir, ".tfenvgo")
	terraformBinPath = filepath.Join(rootURL, "bin")
	terraformVersionPath = filepath.Join(rootURL, "versions")
	currentTerraformVersionPath = filepath.Join(terraformBinPath, "terraform")

	// One-time migration: if versionsRoot contains semver dirs directly (old layout),
	// move them under versions/terraform/.
	if err := migrateVersionsLayout(); err != nil {
		LogWarn("layout migration warning: %v", err)
	}

	return nil
}

// migrateVersionsLayout detects the old flat layout and moves entries under versions/terraform/.
func migrateVersionsLayout() error {
	entries, err := os.ReadDir(terraformVersionPath)
	if err != nil {
		// Directory may not exist yet — not an error.
		return nil
	}

	tfDir := versionsDir(FlavorTerraform)

	for _, e := range entries {
		name := e.Name()
		// Skip flavor subdirs that already exist
		if name == FlavorTerraform || name == FlavorTofu {
			continue
		}
		// If the entry looks like a version directory (contains terraform binary), migrate it.
		oldPath := filepath.Join(terraformVersionPath, name)
		tfBin := filepath.Join(oldPath, "terraform")
		if _, statErr := os.Stat(tfBin); statErr != nil {
			continue // not an old-style version dir
		}

		if err := os.MkdirAll(tfDir, 0o750); err != nil {
			return fmt.Errorf("failed to create %s: %w", tfDir, err)
		}
		newPath := filepath.Join(tfDir, name)
		if _, statErr := os.Stat(newPath); statErr == nil {
			// Already migrated
			continue
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to migrate %s: %w", name, err)
		}
		LogInfo("Migrated %s to %s", oldPath, newPath)
	}
	return nil
}

var (
	rootURL                     string
	terraformBinPath            string
	terraformVersionPath        string
	currentTerraformVersionPath string
	flavorFlag                  string
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
const versionEnvKey = "TFENVGO_VERSION"
const flavorEnvKey = "TFENVGO_FLAVOR"

// Arguments
const (
	latestArg        = "latest"
	latestAllowedArg = "latest-allowed"
	minRequiredArg   = "min-required"
)

const terraformVersionFilename string = ".terraform-version"

// flags
var PreReleaseVersionsIncluded bool
