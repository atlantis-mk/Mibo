package library

import (
	"path"
	"strings"
)

var catalogScanArtworkExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isCatalogScanArtworkFile(storagePath string) bool {
	_, ok := catalogScanArtworkExtensions[strings.ToLower(path.Ext(strings.TrimSpace(storagePath)))]
	return ok
}
