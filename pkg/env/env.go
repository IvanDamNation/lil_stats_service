// Package env for simple parsing .env
package env

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}

	return defaultVal
}

func GetEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func GetEnvDuration(key string, defaultVal int) time.Duration {
	valStr := os.Getenv(key)
	if val, err := strconv.Atoi(valStr); err == nil {
		return time.Duration(val) * time.Second
	}
	return time.Duration(defaultVal) * time.Second
}

func LoadEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			os.Setenv(key, val)
		}
	}

	return scanner.Err()
}
