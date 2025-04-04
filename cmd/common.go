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
		file, err := os.Open(filepath.Join(cwd, entry.Name()))
		if err != nil {
			return "", err
		}
		defer file.Close()

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
			return "", err
		}
	}

	if requiredVersion == "" {
		return "", fmt.Errorf("required_version not found")
	}

	return requiredVersion, nil
}

func getMinRequired(target string) (string, error) {
	terraformVersionContraint, _ := getTerraformVersionConstraint()
	fmt.Println("Found version constraint: " + terraformVersionContraint)
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
	fmt.Println("Found version constraint: " + terraformVersionContraint)
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
		fmt.Println(Red + "Invalid version provided. Allowed values are: " + strings.Join(validArgs, ", ") + " or a valid semver version" + Reset)
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
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Scan file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := terraformVersionRegex.FindStringSubmatch(line); matches != nil {
			terraformVersion := matches[0]
			return terraformVersion, err // Stop walking once we find the version, so it will be only first match
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", err
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
