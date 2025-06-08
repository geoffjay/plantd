// Package util provides utility functions.
package util

import "os"

// Getenv retrieves an environment variable with a fallback value.
func Getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
