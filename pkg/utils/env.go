// Package utils provides environment variable utilities.
package utils

import (
	"fmt"
	"os"
	"strings"
)

// LoadEnv loads .env file and exports variables to environment.
// If .env file doesn't exist, it silently returns (uses system env vars).
func LoadEnv() {
	envFile := ".env"
	data, err := os.ReadFile(envFile)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			_ = os.Setenv(key, value)
		}
	}
	fmt.Printf("Loaded environment from %s\n", envFile)
}

// GetEnv returns env value or fallback.
func GetEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetEnvInt returns env int value or fallback.
func GetEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		_, err := fmt.Sscanf(v, "%d", &i)
		if err == nil && i > 0 {
			return i
		}
	}
	return fallback
}
