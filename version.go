package main

import (
	"strings"
)

// Version stores the version tag - Should include leading 'v' - Update before tagging new versions.
//
var Version = "v0.6.5"

// BuildDate is optional and can be set using '-ldflags "-X 'main.BuildDate=..."'.
//
var BuildDate string

// GitSummary is optional and can be set using '-ldflags "-X 'main.BuildDate=..."'.
//
var GitSummary string

// versionString generates a version string from available vars.
// Variable names are compatible with govvv
//
func versionString() string {
	version := strings.Builder{}

	// Version
	//
	version.WriteString(Version)

	// BuildDate / GitSummary
	//
	if len(BuildDate) > 0 || len(GitSummary) > 0 {
		if version.Len() > 0 {
			version.WriteString(" ")
		}
		version.WriteString("(")
		// Git Summary
		//
		if len(GitSummary) > 0 {
			version.WriteString("build=")
			version.WriteString(GitSummary)
		}
		// Build Date
		//
		if len(BuildDate) > 0 {
			if len(GitSummary) > 0 {
				version.WriteString(" ")
			}
			version.WriteString("date=")
			version.WriteString(BuildDate)
		}
		version.WriteString(")")
	}
	return version.String()
}
