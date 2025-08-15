package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
		file, err := os.Open(filePath)
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

func getMinRequired(target string) (string, error) {
	terraformVersionContraint, _ := getTerraformVersionConstraint()
	LogInfo("Found version constraint: %s", terraformVersionContraint)
	constraints, err := semver.NewConstraint(terraformVersionContraint)
	if err != nil {
		return "", fmt.Errorf("invalid constraint: %w", err)
	}

	var terraformVersions []string
	switch target {
	case "local":
		terraformVersions, _ = getLocalTerraformVersions(false)

	case "remote":
		terraformVersions, _ = getRemoteTerraformVersions(false)
	}

	if len(terraformVersions) == 0 {
		return "", fmt.Errorf("no terraform versions found")
	}

	var validVersions []*semver.Version
	for _, versionStr := range terraformVersions {
		version, err := semver.NewVersion(versionStr)
		if err != nil {
			continue // Skip invalid versions
		}
		if constraints.Check(version) {
			validVersions = append(validVersions, version)
		}
	}

	if len(validVersions) == 0 {
		return "", fmt.Errorf("no available versions satisfy the constraint")
	}

	sort.Sort(semver.Collection(validVersions))

	return validVersions[0].String(), nil // Return the smallest matching version
}

func getLatestAllowed(target, constraint string) (string, error) {
	var terraformVersionContraint string
	if constraint == "" {
		terraformVersionContraint, _ = getTerraformVersionConstraint()
	} else {
		terraformVersionContraint = constraint
	}
	LogInfo("Found version constraint: %s", terraformVersionContraint)
	constraints, err := semver.NewConstraint(terraformVersionContraint)
	if err != nil {
		return "", fmt.Errorf("invalid constraint: %w", err)
	}

	var terraformVersions []string
	switch target {
	case "local":
		terraformVersions, _ = getLocalTerraformVersions(false)

	case "remote":
		terraformVersions, _ = getRemoteTerraformVersions(false)
	}

	if len(terraformVersions) == 0 {
		return "", fmt.Errorf("no terraform versions found")
	}

	var validVersions []*semver.Version
	for _, versionStr := range terraformVersions {
		version, err := semver.NewVersion(versionStr)
		if err != nil {
			continue // Skip invalid versions
		}
		if constraints.Check(version) {
			validVersions = append(validVersions, version)
		}
	}

	if len(validVersions) == 0 {
		return "", fmt.Errorf("no available versions satisfy the constraint")
	}

	sort.Sort(sort.Reverse(semver.Collection(validVersions))) // Even though getRemoteTerraformVersions() returns versions in desc order, sort it to ensure it

	return validVersions[0].String(), nil // Return the highest matching version
}

func validateArg(arg string, allowedVersions map[string]bool) error {
	if allowedVersions[arg] {
		return nil
	}

	// Check if it's a valid Semver version
	if _, err := semver.NewVersion(arg); err != nil {
		validArgs := make([]string, 0, len(allowedVersions))
		for k := range allowedVersions {
			validArgs = append(validArgs, k)
		}
		LogError("Invalid version provided. Allowed values are: %s or a valid semver version", strings.Join(validArgs, ", "))
		return err
	}
	return nil
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
	file, err := os.Open(path)
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

func getCurrentTerraformVersion() (string, error) {
	currentTerraformBinPath, err := os.Readlink(currentTerraformVersionPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink to current terraform version")
	}
	currentTerraformVersionPath := strings.Split(currentTerraformBinPath, "/")
	currentTerraforVersion := currentTerraformVersionPath[len(currentTerraformVersionPath)-2]

	return currentTerraforVersion, err
}
