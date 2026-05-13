package git

var (
	// gitConfig is the package-level singleton used by GetGHUsername.
	// Tests may override via SetUsernameOverride.
	gitConfig = &Config{}
)

// Config holds test-overridable state for Git / GitHub interactions.
type Config struct {
	usernameOverride *string
}

// SetUsernameOverride sets a fixed username to return from GetGHUsername.
// Pass nil to clear.
func (c *Config) SetUsernameOverride(u *string) {
	c.usernameOverride = u
}

// UsernameOverride returns the current override, or nil.
func (c *Config) UsernameOverride() *string {
	return c.usernameOverride
}

// DefaultConfig returns the package-level singleton.
func DefaultConfig() *Config { return gitConfig }
