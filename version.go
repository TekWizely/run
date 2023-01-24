package main

import (
	"strings"
)

// Version stores the version tag - Should include leading 'v' - Update before tagging new versions.
//
var Version = "v0.11.1"

// BuildDate is optional and can be set using '-ldflags "-X 'main.BuildDate=..."'.
//
var BuildDate string

// GitSummary is optional and can be set using '-ldflags "-X 'main.GitSummary=..."'.
// Generally meant to contain the value of:
//   git describe --tags --dirty --always
//
var GitSummary string

// BuildTool is optional and can be set using '-ldflags "-X 'main.BuildTool=..."'.
var BuildTool string

// versionString generates a version string from available vars.
// Variable names are compatible with govvv where applicable
//
func versionString() string {
	version := strings.Builder{}

	// Version
	//
	version.WriteString(Version)

	// Extras
	//
	if len(BuildDate) > 0 || len(GitSummary) > 0 || len(BuildTool) > 0 {
		if version.Len() > 0 {
			version.WriteString(" ")
		}
		version.WriteString("(")
		needsSpace := false
		// Git Summary
		//
		if len(GitSummary) > 0 {
			version.WriteString("build=")
			version.WriteString(GitSummary)
			needsSpace = true
		}
		// Build Date
		//
		if len(BuildDate) > 0 {
			if needsSpace {
				version.WriteString(" ")
			}
			version.WriteString("date=")
			version.WriteString(BuildDate)
			needsSpace = true
		}
		// Build Tool
		//
		if len(BuildTool) > 0 {
			if needsSpace {
				version.WriteString(" ")
			}
			version.WriteString("builder=")
			version.WriteString(BuildTool)
			// needsSpace = true
		}
		version.WriteString(")")
	}
	return version.String()
}
