package util

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

// ReadFileIfExists returns contents of file at specified path as a byte array,
// while letting you provide specific NotExist error handling
// Returns []byte, true, nil if file exists, is normal, and can be read
// Returns nil, false, nil if err == fs.ErrNotExist
// Returns nil, false, err if err && err != fs.ErrNotExist
//
func ReadFileIfExists(path string) ([]byte, bool, error) {
	var (
		stat   os.FileInfo
		exists bool
		err    error
		file   *os.File
		bytes  []byte
	)

	if stat, exists, err = StatIfExists(path); !exists {
		return nil, exists, err
	}
	if !stat.Mode().IsRegular() {
		return nil, false, fmt.Errorf("file '%s': file not considered 'regular'", path)
	}
	// Open the file
	// filePath.Clean to appease the gosec gods [G304 (CWE-22)]
	//
	if file, err = os.Open(filepath.Clean(path)); err == nil {
		// Close file before we exit
		//
		defer func() { _ = file.Close() }()
		// Read file into memory
		//
		if bytes, err = ioutil.ReadAll(file); err == nil {
			return bytes, true, nil
		}
	}
	// If we get here, we have an error
	//
	return nil, false, err
}

// TryMakeRelative tries to generate a path for targetPath that is relative to basePath.
// It returns either a path relative to basePath, if possible, or targetPath.
//
func TryMakeRelative(basePath string, targetPath string) string {
	if rel, err := filepath.Rel(basePath, targetPath); err == nil && len(rel) > 0 && !strings.HasPrefix(rel, ".") {
		return rel
	}
	return targetPath
}
