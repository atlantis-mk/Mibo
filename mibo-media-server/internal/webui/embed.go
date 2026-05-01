package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embeddedDist embed.FS

func EmbeddedDist() fs.FS {
	dist, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return embeddedDist
	}
	return dist
}
