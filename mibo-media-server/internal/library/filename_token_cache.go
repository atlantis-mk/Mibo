package library

import "strings"

type filenameTokenProfileCache struct {
	profilesByPath map[string]filenameSignalModel
	counters       *contentShapeCounters
}

func newFilenameTokenProfileCache() *filenameTokenProfileCache {
	return &filenameTokenProfileCache{profilesByPath: make(map[string]filenameSignalModel)}
}

func filenameTokenProfileForPath(cache *filenameTokenProfileCache, storagePath string) filenameSignalModel {
	pathKey := strings.TrimSpace(storagePath)
	if cache == nil {
		return extractFilenameSignalModel(pathKey)
	}
	if profile, ok := cache.profilesByPath[pathKey]; ok {
		return profile
	}
	profile := extractFilenameSignalModel(pathKey)
	if cache.counters != nil {
		cache.counters.TokenProfileParses++
	}
	cache.profilesByPath[pathKey] = profile
	return profile
}
