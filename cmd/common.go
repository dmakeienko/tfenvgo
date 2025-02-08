package cmd

import "os"

func getEnv(envVar, defaultValue string) string {
	if envVarValue, envVarPresent := os.LookupEnv(envVar); envVarPresent {
		return envVarValue
	}
	return defaultValue
}
