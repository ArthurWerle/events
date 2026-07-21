// Package assets embeds static UI assets (fonts, etc.) into the binary so the
// service is self-contained and can serve them from memory at runtime.
package assets

import "embed"

// FS holds the embedded static assets. Files are served under /static, e.g.
// the file fonts/GeistVF.woff is reachable at /static/fonts/GeistVF.woff.
//
//go:embed fonts/*.woff
var FS embed.FS
