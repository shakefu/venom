package venom

import (
	"runtime/debug"
	"time"
)

// resolveEffectiveVersion implements rule ResolveAppVersion in venom.allium.
// The version reported by --version is resolved in priority order:
//
//  1. explicit (set via WithVersion).
//  2. mainVersion from runtime/debug.ReadBuildInfo, if non-empty and not "(devel)".
//  3. First 7 chars of vcsRevision, if non-empty.
//  4. "build-" + now formatted as YYYYMMDDhhmm, so the version is never empty.
//
// The pure helper is the testable seam; the runtime wrapper effectiveVersion
// reads the actual build info and the current time and threads them through.
func resolveEffectiveVersion(explicit, mainVersion, vcsRevision string, now time.Time) string {
	if explicit != "" {
		return explicit
	}
	if mainVersion != "" && mainVersion != "(devel)" {
		return mainVersion
	}
	if vcsRevision != "" {
		if len(vcsRevision) > 7 {
			return vcsRevision[:7]
		}
		return vcsRevision
	}
	return "build-" + now.Format("200601021504")
}

// effectiveVersion reads runtime/debug.ReadBuildInfo and resolves the version
// using resolveEffectiveVersion.
func effectiveVersion(explicit string) string {
	var mainVersion, vcsRevision string
	if info, ok := debug.ReadBuildInfo(); ok {
		mainVersion = info.Main.Version
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				vcsRevision = s.Value
				break
			}
		}
	}
	return resolveEffectiveVersion(explicit, mainVersion, vcsRevision, time.Now())
}
