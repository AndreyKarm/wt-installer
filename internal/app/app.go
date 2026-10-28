// Package app holds identifiers shared by the tray app, server and updater.
package app

const (
	// Name is also the auto-start registry value name; changing it would
	// orphan existing users' auto-start entries.
	Name    = "WT Installer Server"
	Version = "0.1.7"
	// Port is also hard-coded in the extension (background.js, popup.js and
	// manifest.json host_permissions). Change them together.
	Port = 4316
)
