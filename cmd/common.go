package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func getEnv(envVar, defaultValue string) string {
	if envVarValue, envVarPresent := os.LookupEnv(envVar); envVarPresent {
		return envVarValue
	}
	return defaultValue
}

func getTerraformVersionConstraint() (string, error) {
	// Define regex pattern to match required_version
	requiredVersionPattern := regexp.MustCompile(`required_version\s*=\s*"([^"]+)"`)

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current directory: %w", err)
	}

	// Read all files in the current directory
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return "", fmt.Errorf("error reading current directory: %w", err)
	}

	var requiredVersion string
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}
		// Only process .tf files
		if filepath.Ext(entry.Name()) != ".tf" {
			continue
		}
		// Open file for reading
		filePath := filepath.Join(cwd, entry.Name())
		// Defensive absolute path check
		absCwd, _ := filepath.Abs(cwd)
		absFile, err := filepath.Abs(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
		if !strings.HasPrefix(absFile, absCwd+string(os.PathSeparator)) && absFile != absCwd {
			return "", fmt.Errorf("file path outside current directory: %s", entry.Name())
		}
		// Path is validated above; safe to open.
		file, err := os.Open(filePath) // #nosec G304 - path is validated above
		if err != nil {
			return "", fmt.Errorf("failed to open file %s: %w", entry.Name(), err)
		}
		// Ensure file is closed and log any close errors
		defer func() {
			if cerr := file.Close(); cerr != nil {
				LogWarn("failed to close file %s: %v", filePath, cerr)
			}
		}()

		// Scan file line by line
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if matches := requiredVersionPattern.FindStringSubmatch(line); matches != nil {
				requiredVersion = matches[1]
				return requiredVersion, nil // Stop once we find the required version
			}
		}

		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error scanning file %s: %w", entry.Name(), err)
		}
	}

	if requiredVersion == "" {
		return "", fmt.Errorf("required_version not found in any .tf files")
	}

	return requiredVersion, nil
}

// Sources that version keywords are resolved against.
const (
	targetLocal  = "local"
	targetRemote = "remote"
)

// candidateVersions lists the versions available from target (installed or
// remote), parsed and sorted newest first.
func candidateVersions(flv, target string, includePrerelease bool) ([]*semver.Version, error) {
	var (
		raw []string
		err error
	)
	switch target {
	case targetLocal:
		raw, err = getLocalVersions(flv, includePrerelease)
	case targetRemote:
		raw, err = getRemoteVersions(flv, includePrerelease)
	default:
		return nil, fmt.Errorf("unknown version source %q", target)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list %s %s versions: %w", target, flv, err)
	}

	versions := make([]*semver.Version, 0, len(raw))
	for _, s := range raw {
		v, err := semver.NewVersion(s)
		if err != nil {
			continue // Skip invalid versions
		}
		versions = append(versions, v)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("no %s %s versions found", target, flv)
	}

	sort.Sort(sort.Reverse(semver.Collection(versions)))
	return versions, nil
}

// requiredVersionConstraint parses required_version from the *.tf files in
// the current directory.
func requiredVersionConstraint() (*semver.Constraints, error) {
	raw, err := getTerraformVersionConstraint()
	if err != nil {
		return nil, err
	}
	LogInfo("Found version constraint: %s", raw)
	constraints, err := semver.NewConstraint(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid required_version constraint %q: %w", raw, err)
	}
	return constraints, nil
}

// getMinRequired returns the oldest version from target that satisfies required_version.
func getMinRequired(flv, target string) (string, error) {
	constraints, err := requiredVersionConstraint()
	if err != nil {
		return "", err
	}
	versions, err := candidateVersions(flv, target, false)
	if err != nil {
		return "", err
	}
	for i := len(versions) - 1; i >= 0; i-- {
		if constraints.Check(versions[i]) {
			return versions[i].String(), nil
		}
	}
	return "", fmt.Errorf("no %s %s version satisfies the constraint", target, flv)
}

// getLatestAllowed returns the newest version from target that satisfies required_version.
func getLatestAllowed(flv, target string) (string, error) {
	constraints, err := requiredVersionConstraint()
	if err != nil {
		return "", err
	}
	versions, err := candidateVersions(flv, target, false)
	if err != nil {
		return "", err
	}
	for _, v := range versions {
		if constraints.Check(v) {
			return v.String(), nil
		}
	}
	return "", fmt.Errorf("no %s %s version satisfies the constraint", target, flv)
}

// getLatestMatching returns the newest version from target whose string form
// matches the Go regular expression pattern.
func getLatestMatching(flv, target, pattern string, includePrerelease bool) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	versions, err := candidateVersions(flv, target, includePrerelease)
	if err != nil {
		return "", err
	}
	for _, v := range versions {
		if re.MatchString(v.String()) {
			return v.String(), nil
		}
	}
	return "", fmt.Errorf("no %s %s version matches regex %q", target, flv, pattern)
}

// parseVersionArgs splits CLI arguments into a requested version (a semver
// string or keyword) and an optional regex. With no arguments it uses
// fallback(), and "latest" if that is empty.
func parseVersionArgs(args []string, fallback func() string) (requested, pattern string, err error) {
	switch len(args) {
	case 0:
		requested = fallback()
		if requested == "" {
			requested = latestArg
		}
		return requested, "", nil
	case 1:
		return args[0], "", nil
	default:
		if args[0] != latestArg {
			return "", "", fmt.Errorf("a regex can only follow %q, got %q", latestArg, args[0])
		}
		return latestArg, args[1], nil
	}
}

// versionFromEnvOrFile is the default version for install and use:
// TFENVGO_VERSION, then TFENVGO_TERRAFORM_VERSION, then .terraform-version.
func versionFromEnvOrFile() string {
	versionFromFile, _ := readVersionFromFile()
	return getEnv(versionEnvKey, getEnv(terraformVersionEnvKey, versionFromFile))
}

// resolveVersion turns a requested version (an exact semver version or one of
// the allowed keywords) into a concrete version, looking it up in target.
func resolveVersion(flv, requested, pattern, target string, keywords []string, includePrerelease bool) (string, error) {
	if !slices.Contains(keywords, requested) {
		if _, err := semver.NewVersion(requested); err != nil {
			return "", fmt.Errorf("invalid version %q: expected a semver version or one of: %s", requested, strings.Join(keywords, ", "))
		}
		return requested, nil
	}

	switch requested {
	case latestArg:
		if pattern != "" {
			return getLatestMatching(flv, target, pattern, includePrerelease)
		}
		versions, err := candidateVersions(flv, target, includePrerelease)
		if err != nil {
			return "", err
		}
		return versions[0].String(), nil
	case latestAllowedArg:
		return getLatestAllowed(flv, target)
	case minRequiredArg:
		return getMinRequired(flv, target)
	default:
		return "", fmt.Errorf("unsupported version keyword %q", requested)
	}
}

func readVersionFromFile() (string, error) {
	// Get current directory
	terraformVersionRegex := regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current directory: %w", err)
	}
	path := filepath.Join(cwd, terraformVersionFilename)

	// Open file for reading
	// Ensure the path is inside current working directory
	if !strings.HasPrefix(path, cwd) {
		// defensive: construct with Join
		path = filepath.Join(cwd, terraformVersionFilename)
	}
	// Path is validated above; safe to open.
	file, err := os.Open(path) // #nosec G304 - path is constructed from cwd + known filename
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", terraformVersionFilename, err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			LogWarn("failed to close %s: %v", terraformVersionFilename, cerr)
		}
	}()

	// Scan file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := terraformVersionRegex.FindStringSubmatch(line); matches != nil {
			terraformVersion := matches[0]
			return terraformVersion, nil // Stop walking once we find the version, so it will be only first match
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning %s: %w", terraformVersionFilename, err)
	}

	return "", fmt.Errorf("no valid version found in %s", terraformVersionFilename)
}

// getCurrentVersion reads the symlink for the given flavor and returns the active version string.
func getCurrentVersion(flv string) (string, error) {
	symlinkPath := currentSymlinkPath(flv)
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink to current %s version", flv)
	}
	parts := strings.Split(target, string(filepath.Separator))
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected symlink target format: %s", target)
	}
	// The version dir is the second-to-last component: .../versions/<flavor>/<version>/<binary>
	return parts[len(parts)-2], nil
}
