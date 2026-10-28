// Package media embeds the app icon. It lives next to icon.ico because
// go:embed can't reference files outside the package directory.
package media

import _ "embed"

//go:embed icon.ico
var Icon []byte
