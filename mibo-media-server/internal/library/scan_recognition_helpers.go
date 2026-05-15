package library

import (
	"path"
	"strings"
)

const LibraryTypeAuto = "auto"

func isVideoFile(itemPath string) bool {
	_, ok := videoExtensions[strings.ToLower(path.Ext(itemPath))]
	return ok
}

func normalizeLibraryType(libraryType string) string {
	switch strings.ToLower(strings.TrimSpace(libraryType)) {
	case "", "auto", "source", "source-first", "source_first", "movie", "movies", "films", "tv", "tvshows", "shows", "mixed", "mixed-content", "mixed_content":
		return LibraryTypeAuto
	default:
		return LibraryTypeAuto
	}
}

func effectiveVideoLibraryType(libraryType string) string {
	return normalizeLibraryType(libraryType)
}
