package main

import (
	"fmt"
	"strings"
)

// Version stores the version tag - Should include leading 'v' - Update before tagging new versions.
//
var Version = "v0.6.4"

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

	version.WriteString(Version)
	if len(BuildDate) > 0 || len(GitSummary) > 0 {
		// Version
		//
		if version.Len() > 0 {
			version.WriteString(" ")
		}
		version.WriteString("(")
		// Git Summary
		//
		if len(BuildDate) > 0 {
			version.WriteString("build=")
			version.WriteString(GitSummary)
		}
		// Build Date
		//
		if len(BuildDate) > 0 {
			if len(BuildDate) > 0 {
				version.WriteString(" ")
			}
			version.WriteString("date=")
			version.WriteString(BuildDate)
		}
		version.WriteString(")")
	}
	return version.String()
}

func showVersion() {
	fmt.Println("run", versionString())
}
