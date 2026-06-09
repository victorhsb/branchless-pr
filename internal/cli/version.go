package cli

// version is set at build time via -ldflags.
// Default matches the latest tagged release.
var version = "1.10.0"

// Version returns the current CLI version string.
func Version() string {
	return version
}
