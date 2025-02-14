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

	// Walk through all files in the current directory
	var requiredVersion string
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		// Open file for reading
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Scan file line by line
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if matches := requiredVersionPattern.FindStringSubmatch(line); matches != nil {
				requiredVersion = matches[1]
				return nil // Stop walking once we find the required version
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking through files: %w", err)
	}

	if requiredVersion == "" {
		return "", fmt.Errorf("required_version not found")
	}

	return requiredVersion, nil
}

func getMinRequired() (string, error) {
	terraformVersionContraint, _ := getTerraformVersionConstraint()
	fmt.Println("Found version constraint: " + terraformVersionContraint)
	constraints, err := semver.NewConstraint(terraformVersionContraint)

	terraformVersions, _ := getTerraformVersions()

	if len(terraformVersions) == 0 {
		return "", fmt.Errorf("no terraform versions found")
	}

	if err != nil {
		return "", fmt.Errorf("invalid constraint: %w", err)
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

func getLatestAllowed() (string, error) {
	terraformVersionContraint, _ := getTerraformVersionConstraint()
	fmt.Println("Found version constraint: " + terraformVersionContraint)
	constraints, err := semver.NewConstraint(terraformVersionContraint)

	terraformVersions, _ := getTerraformVersions()

	if len(terraformVersions) == 0 {
		return "", fmt.Errorf("no terraform versions found")
	}

	if err != nil {
		return "", fmt.Errorf("invalid constraint: %w", err)
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

	sort.Sort(sort.Reverse(semver.Collection(validVersions))) // Even though getTerraformVersions() returns versions in desc order, sort it to ensure it

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
