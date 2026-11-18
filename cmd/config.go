package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	FlavorTerraform = "terraform"
	FlavorTofu      = "tofu"
)

// Upstream release locations. Variables rather than constants so tests can
// point them at a local httptest server.
var (
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

// validateFlavor rejects anything but the known flavors. The flavor becomes a
// path component under ~/.tfenvgo/versions, so an arbitrary value such as
// "../.." would let install and uninstall reach outside that directory.
func validateFlavor(flv string) error {
	if flv != FlavorTerraform && flv != FlavorTofu {
		return fmt.Errorf("invalid flavor %q, allowed values: %s, %s", flv, FlavorTerraform, FlavorTofu)
	}
	return nil
}

// flavorSource returns the effective flavor and a description of where it came from.
// Precedence: --flavor flag > TFENVGO_FLAVOR env > ~/.tfenvgo/flavor file > terraform default.
func flavorSource(flag string) (flavor, source string, err error) {
	if flag != "" {
		if err := validateFlavor(flag); err != nil {
			return "", "", fmt.Errorf("--flavor: %w", err)
		}
		return flag, "--flavor flag", nil
	}
	if env := os.Getenv(flavorEnvKey); env != "" {
		if err := validateFlavor(env); err != nil {
			return "", "", fmt.Errorf("%s: %w", flavorEnvKey, err)
		}
		return env, flavorEnvKey + " env var", nil
	}
	flavorFile := filepath.Join(rootURL, "flavor")
	data, err := os.ReadFile(flavorFile) // #nosec G304 - path is constructed from controlled rootURL
	if err == nil {
		f := strings.TrimSpace(string(data))
		if validateFlavor(f) == nil {
			return f, flavorFile, nil
		}
		LogWarn("ignoring invalid flavor %q in %s", f, flavorFile)
	}
	return FlavorTerraform, "built-in default", nil
}

// resolveFlavor returns the effective, validated flavor for this invocation.
func resolveFlavor(flag string) (string, error) {
	flv, _, err := flavorSource(flag)
	return flv, err
}

// validatePlatform checks the OS and architecture used in download URLs and
// archive names, so a typo in TFENVGO_OS_TYPE or TFENVGO_ARCH fails clearly.
func validatePlatform(osType, arch string) error {
	for name, value := range map[string]string{osTypeEnvKey: osType, archEnvKey: arch} {
		if !platformRegex.MatchString(value) {
			return fmt.Errorf("invalid %s value %q: expected something like linux, darwin, amd64 or arm64", name, value)
		}
	}
	return nil
}

var platformRegex = regexp.MustCompile(`^[a-z0-9]+$`)

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
	rootURL              string
	terraformBinPath     string
	terraformVersionPath string
	flavorFlag           string
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

func init() {
	if !colorEnabled() {
		Reset, Red, Green, Yellow, Blue, Magenta, Cyan, Gray, White = "", "", "", "", "", "", "", "", ""
	}
}

// colorEnabled reports whether ANSI colors should be used: not when NO_COLOR
// is set (https://no-color.org), TERM is "dumb", or stdout is not a terminal
// (pipes, files, CI logs).
func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// Environment variables
const archEnvKey = "TFENVGO_ARCH"
const osTypeEnvKey = "TFENVGO_OS_TYPE"
const terraformVersionEnvKey = "TFENVGO_TERRAFORM_VERSION"
const versionEnvKey = "TFENVGO_VERSION"
const flavorEnvKey = "TFENVGO_FLAVOR"
const downloadTimeoutEnvKey = "TFENVGO_DOWNLOAD_TIMEOUT"

// defaultDownloadTimeout is used when TFENVGO_DOWNLOAD_TIMEOUT is unset or invalid.
const defaultDownloadTimeout = 300 * time.Second

// downloadTimeout returns the HTTP timeout used when downloading binaries.
// It is read at runtime from TFENVGO_DOWNLOAD_TIMEOUT, which accepts either a
// plain number of seconds ("600") or a Go duration string ("10m", "90s").
// Invalid or non-positive values fall back to the default.
func downloadTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(downloadTimeoutEnvKey))
	if raw == "" {
		return defaultDownloadTimeout
	}

	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			LogWarn("Invalid %s value %q: must be positive, using %s", downloadTimeoutEnvKey, raw, defaultDownloadTimeout)
			return defaultDownloadTimeout
		}
		return time.Duration(seconds) * time.Second
	}

	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		LogWarn("Invalid %s value %q: expected seconds or a duration like 10m, using %s", downloadTimeoutEnvKey, raw, defaultDownloadTimeout)
		return defaultDownloadTimeout
	}
	return duration
}

// Arguments
const (
	latestArg        = "latest"
	latestAllowedArg = "latest-allowed"
	minRequiredArg   = "min-required"
)

const terraformVersionFilename string = ".terraform-version"

// flags
var PreReleaseVersionsIncluded bool
