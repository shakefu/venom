package venom

import (
	"strings"
	"unicode"
)

// funcNameToCommandPath splits a function name on underscores to produce
// a command hierarchy. A package prefix (e.g. "main.") is stripped first.
// Each segment is converted from camelCase to kebab-case.
//
//	"serve_tls"        → ["serve", "tls"]
//	"main.serve"       → ["serve"]
//	"initProject"      → ["init-project"]
//	"init_fastProject" → ["init", "fast-project"]
func funcNameToCommandPath(name string) []string {
	// Strip package prefix if present.
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}

	if name == "" {
		return nil
	}

	parts := strings.Split(name, "_")
	// Filter out empty segments from leading/trailing/consecutive underscores,
	// and convert each segment from camelCase to kebab-case.
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, paramToFlagName(p))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// paramToFlagName converts a camelCase parameter name to kebab-case.
//
//	"serverPort" → "server-port"
//	"port"       → "port"
//	"TLSConfig"  → "tls-config"
func paramToFlagName(param string) string {
	if param == "" {
		return ""
	}

	var b strings.Builder
	runes := []rune(param)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Insert a hyphen before an uppercase letter when:
			// - It's not the first character, AND
			// - The previous character is lowercase, OR
			// - The next character is lowercase (handles "TLSConfig" → "tls-config").
			if i > 0 {
				prevLower := unicode.IsLower(runes[i-1])
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if prevLower || nextLower {
					b.WriteByte('-')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// flagToEnvVar converts a kebab-case flag name to SCREAMING_SNAKE_CASE
// with the given prefix.
//
//	prefix="MYAPP", flag="server-port" → "MYAPP_SERVER_PORT"
func flagToEnvVar(prefix, flag string) string {
	upper := strings.ToUpper(strings.ReplaceAll(flag, "-", "_"))
	if prefix == "" {
		return upper
	}
	return prefix + "_" + upper
}

// flagToConfigKey returns the configuration file key for a flag name.
// For now this is an identity transform.
func flagToConfigKey(flag string) string {
	return flag
}

// deriveAppName returns the last path segment of a Go module path.
//
//	"github.com/shakefu/venom" → "venom"
func deriveAppName(modulePath string) string {
	if modulePath == "" {
		return ""
	}
	if idx := strings.LastIndex(modulePath, "/"); idx >= 0 {
		return modulePath[idx+1:]
	}
	return modulePath
}
