package util

import (
	"errors"
	"io/fs"
	"os"
)

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

// StatIfExists lets you provide specific NotExist error handling
// Returns stat, true, nil if file exists
// Returns nil, false, nil if err == fs.ErrNotExist
// Returns nil, false, err if err && err != fs.ErrNotExist
//
func StatIfExists(path string) (os.FileInfo, bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		return stat, true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, err
}
