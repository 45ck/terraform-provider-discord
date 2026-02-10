package discord

import "fmt"

var buildVersion = "dev"

// SetBuildVersion sets the version string used in diagnostics and the REST User-Agent.
// Call this from main() before serving the provider.
func SetBuildVersion(v string) {
	if v == "" {
		return
	}
	buildVersion = v
}

func userAgent() string {
	// Keep this stable and parseable for Discord support/debugging.
	// Example: terraform-provider-discord/0.1.2 (45ck fork)
	return fmt.Sprintf("terraform-provider-discord/%s (45ck fork)", buildVersion)
}
