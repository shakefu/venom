package venom

import (
	"testing"
	"time"
)

// TestResolveEffectiveVersion verifies the version fallback chain described by
// rule ResolveAppVersion in venom.allium:
//
//  1. Explicit App.version (set via WithVersion).
//  2. runtime/debug.ReadBuildInfo().Main.Version, if set and not "(devel)".
//  3. Short VCS revision (first 7 chars), if VCS info is embedded.
//  4. "build-" + current timestamp formatted as YYYYMMDDhhmm.
//
// The pure helper resolveEffectiveVersion(explicit, mainVersion, vcsRevision, now)
// is the testable seam; the App-level wiring reads runtime/debug.ReadBuildInfo()
// and time.Now() and threads them through this helper.
func TestResolveEffectiveVersion(t *testing.T) {
	fixedNow := time.Date(2026, 4, 30, 14, 35, 0, 0, time.UTC)
	expectedStamp := "build-202604301435"

	tests := []struct {
		name        string
		explicit    string
		mainVersion string
		vcsRevision string
		now         time.Time
		want        string
	}{
		{
			name:        "explicit_wins_over_all",
			explicit:    "v1.2.3",
			mainVersion: "v9.9.9",
			vcsRevision: "abcdef0123456789",
			now:         fixedNow,
			want:        "v1.2.3",
		},
		{
			name:        "main_version_used_when_explicit_empty",
			explicit:    "",
			mainVersion: "v0.5.1",
			vcsRevision: "abcdef0123456789",
			now:         fixedNow,
			want:        "v0.5.1",
		},
		{
			name:        "devel_main_version_is_skipped",
			explicit:    "",
			mainVersion: "(devel)",
			vcsRevision: "abcdef0123456789",
			now:         fixedNow,
			want:        "abcdef0",
		},
		{
			name:        "vcs_revision_truncated_to_seven",
			explicit:    "",
			mainVersion: "",
			vcsRevision: "abcdef0123456789",
			now:         fixedNow,
			want:        "abcdef0",
		},
		{
			name:        "vcs_revision_shorter_than_seven_is_used_as_is",
			explicit:    "",
			mainVersion: "",
			vcsRevision: "abc12",
			now:         fixedNow,
			want:        "abc12",
		},
		{
			name:        "timestamp_fallback_when_all_empty",
			explicit:    "",
			mainVersion: "",
			vcsRevision: "",
			now:         fixedNow,
			want:        expectedStamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveEffectiveVersion(tt.explicit, tt.mainVersion, tt.vcsRevision, tt.now)
			if got != tt.want {
				t.Errorf("resolveEffectiveVersion(%q, %q, %q, %v) = %q, want %q",
					tt.explicit, tt.mainVersion, tt.vcsRevision, tt.now, got, tt.want)
			}
		})
	}
}

// TestEffectiveVersionNeverEmpty asserts the spec guarantee: even with no
// explicit version, no build info, and no VCS revision, the resolved version
// is non-empty.
func TestEffectiveVersionNeverEmpty(t *testing.T) {
	got := resolveEffectiveVersion("", "", "", time.Now())
	if got == "" {
		t.Fatal("resolveEffectiveVersion returned empty string; spec requires version is never empty")
	}
}
