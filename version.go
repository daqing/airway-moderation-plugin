package moderationplugin

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var version string

// Version returns the plugin version, read from the VERSION file.
func Version() string { return strings.TrimSpace(version) }
