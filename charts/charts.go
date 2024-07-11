package charts

import (
	"embed"
)

// InternalChart embeds the internal charts in embed.FS
//
//go:embed internal
var InternalChart embed.FS

// InternalChartsPath is the path to the internal charts
const InternalChartsPath = "internal"
