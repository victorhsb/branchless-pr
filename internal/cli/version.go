package cli

// version is set at build time via -ldflags.
// Default matches the original tool's fallback (SPEC §1.2).
var version = "0.1.0"

// Version returns the current CLI version string.
func Version() string {
	return version
}
