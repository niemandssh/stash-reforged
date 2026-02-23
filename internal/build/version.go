// Package build provides the version information for the application.
package build

import (
	"os"
	"regexp"
)

var version string
var buildstamp string
var githash string
var officialBuild string
var versionCheckRepo string

func Version() (string, string, string) {
	return version, githash, buildstamp
}

func VersionString() string {
	var versionString string
	switch {
	case version != "":
		if githash != "" && !IsDevelop() {
			versionString = version + " (" + githash + ")"
		} else {
			versionString = version
		}
	case githash != "":
		versionString = githash
	default:
		versionString = "unknown"
	}
	if IsOfficial() {
		versionString += " - Official Build"
	} else {
		versionString += " - Unofficial Build"
	}
	if buildstamp != "" {
		versionString += " - " + buildstamp
	}
	return versionString
}

func IsOfficial() bool {
	return officialBuild == "true"
}

// VersionCheckRepo returns the GitHub "owner/repo" used for "new version" checks.
// Priority: env STASH_VERSION_CHECK_REPO (if set) → build-time VERSION_CHECK_REPO → default niemandssh/stash-reforged.
// Set STASH_VERSION_CHECK_REPO= to empty to disable the check at runtime.
func VersionCheckRepo() string {
	const defaultRepo = "niemandssh/stash-reforged"
	if v, ok := os.LookupEnv("STASH_VERSION_CHECK_REPO"); ok {
		return v
	}
	if versionCheckRepo != "" {
		return versionCheckRepo
	}
	return defaultRepo
}

func IsDevelop() bool {
	if githash == "" {
		return false
	}

	// if the version is suffixed with -x-xxxx, then we are running a development build
	develop := false
	re := regexp.MustCompile(`-\d+-g\w+$`)
	if re.MatchString(version) {
		develop = true
	}
	return develop
}
