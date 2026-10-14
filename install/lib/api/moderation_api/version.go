package moderation_api

// version is the plugin version the health endpoint reports. The plugin root
// package sets it on init; this package cannot read it directly without
// importing its own parent.
var version = "unknown"

// SetVersion records the plugin version.
func SetVersion(value string) { version = value }
