package util

import "os"

// DefaultIfEmpty returns default string of src string is empty.
//
func DefaultIfEmpty(src string, def string) string {
	if len(src) > 0 {
		return src
	}
	return def
}

// GetEnvOrDefault returns default string if env key empty, as returned
// by os.GetEnv.
//
func GetEnvOrDefault(key string, def string) string {
	return DefaultIfEmpty(os.Getenv(key), def)
}
