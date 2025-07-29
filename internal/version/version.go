// Package version provides version information for Argus Collector tools
package version

import (
	"fmt"
	"runtime"
)

// Build-time variables that can be set via ldflags
var (
	// Version is the main version number that is being run at the moment
	Version = "0.02"

	// GitCommit is the git sha1 that was compiled. This will be filled in by the compiler
	GitCommit = "unknown"

	// GitBranch is the git branch that was compiled. This will be filled in by the compiler
	GitBranch = "unknown"

	// BuildDate is the date the binary was built
	BuildDate = "unknown"

	// BuildUser is the user who built the binary
	BuildUser = "unknown"
)

// BuildInfo contains version and build information
type BuildInfo struct {
	Version   string
	GitCommit string
	GitBranch string
	BuildDate string
	BuildUser string
	GoVersion string
	Platform  string
}

// GetBuildInfo returns complete build information
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildDate: BuildDate,
		BuildUser: BuildUser,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetVersion returns the version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns a detailed version string
func GetFullVersion() string {
	if GitCommit != "unknown" && len(GitCommit) > 7 {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}

// GetVersionInfo returns formatted version information
func GetVersionInfo(appName string) string {
	info := GetBuildInfo()

	result := fmt.Sprintf("%s version %s", appName, info.Version)

	if info.GitCommit != "unknown" {
		if len(info.GitCommit) > 7 {
			result += fmt.Sprintf(" (commit %s)", info.GitCommit[:7])
		} else {
			result += fmt.Sprintf(" (commit %s)", info.GitCommit)
		}
	}

	if info.GitBranch != "unknown" {
		result += fmt.Sprintf(" on branch %s", info.GitBranch)
	}

	if info.BuildDate != "unknown" {
		result += fmt.Sprintf("\nBuilt: %s", info.BuildDate)
	}

	if info.BuildUser != "unknown" {
		result += fmt.Sprintf(" by %s", info.BuildUser)
	}

	result += fmt.Sprintf("\nGo: %s", info.GoVersion)
	result += fmt.Sprintf("\nPlatform: %s", info.Platform)

	return result
}
