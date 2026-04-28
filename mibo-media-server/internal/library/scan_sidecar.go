package library

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/storage"
)

const maxSidecarMetadataBytes = 256 * 1024

type sidecarIndex struct {
	byBase       map[string][]storage.Object
	folderMeta   []storage.Object
	videoCount   int
	providerName string
}

type sidecarMatch struct {
	object            storage.Object
	extension         string
	associationSource string
}

func buildSidecarIndex(providerName string, objects []storage.Object) sidecarIndex {
	index := sidecarIndex{byBase: make(map[string][]storage.Object), providerName: strings.TrimSpace(providerName)}
	for _, object := range objects {
		if object.IsDir {
			continue
		}
		if isVideoFile(object.Path) {
			index.videoCount++
			continue
		}
		ext := sidecarExtension(object.Path)
		if ext == "" {
			continue
		}
		base := sidecarBaseName(object.Path)
		index.byBase[base] = append(index.byBase[base], object)
		if isFolderLevelMetadataName(base, ext) {
			index.folderMeta = append(index.folderMeta, object)
		}
	}
	return index
}

func (idx sidecarIndex) matchesForVideo(videoPath string) []sidecarMatch {
	base := sidecarBaseName(videoPath)
	matches := make([]sidecarMatch, 0, len(idx.byBase[base])+len(idx.folderMeta))
	for _, object := range idx.byBase[base] {
		matches = append(matches, sidecarMatch{object: object, extension: sidecarExtension(object.Path), associationSource: "basename"})
	}
	if idx.videoCount == 1 {
		for _, object := range idx.folderMeta {
			if sidecarBaseName(object.Path) == base {
				continue
			}
			matches = append(matches, sidecarMatch{object: object, extension: sidecarExtension(object.Path), associationSource: "single-video-folder"})
		}
	}
	return matches
}

func sidecarExtension(objectPath string) string {
	switch ext := strings.ToLower(strings.TrimSpace(path.Ext(objectPath))); ext {
	case ".srt", ".ass", ".nfo", ".json":
		return ext
	default:
		return ""
	}
}

func sidecarBaseName(objectPath string) string {
	base := path.Base(strings.TrimSpace(objectPath))
	return strings.ToLower(strings.TrimSuffix(base, path.Ext(base)))
}

func isFolderLevelMetadataName(base string, ext string) bool {
	if ext != ".nfo" && ext != ".json" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "movie", "tvshow", "season", "metadata", "info":
		return true
	default:
		return false
	}
}

func (s *Service) applyCatalogScanSidecars(ctx context.Context, provider storage.Provider, artifact catalogScanArtifact, matches []sidecarMatch) catalogScanArtifact {
	for _, match := range matches {
		sidecar := catalogScanSidecar{Path: strings.TrimSpace(match.object.Path), Extension: match.extension, AssociationSource: match.associationSource, SizeBytes: match.object.Size, ModifiedAt: match.object.Modified, StableIdentityKey: strings.TrimSpace(match.object.StableIdentity)}
		switch match.extension {
		case ".srt", ".ass":
			artifact.SubtitleSidecars = append(artifact.SubtitleSidecars, sidecar)
		case ".nfo", ".json":
			metadata := catalogScanMetadataSidecar{catalogScanSidecar: sidecar, ParseStatus: "skipped"}
			if match.object.Size > maxSidecarMetadataBytes {
				artifact.MetadataSidecars = append(artifact.MetadataSidecars, metadata)
				continue
			}
			content, err := readSidecarMetadataContent(ctx, provider, match.object)
			if err != nil {
				metadata.ParseStatus = "unreadable"
			} else if hints, externalIDs, err := parseSidecarMetadata(match.extension, content); err != nil {
				metadata.ParseStatus = "malformed"
			} else {
				metadata.ParseStatus = "parsed"
				metadata.Hints = hints
				metadata.ExternalIDs = externalIDs
				artifact = applySidecarMetadataHints(artifact, hints)
			}
			artifact.MetadataSidecars = append(artifact.MetadataSidecars, metadata)
		}
	}
	return artifact
}

func readSidecarMetadataContent(ctx context.Context, provider storage.Provider, object storage.Object) ([]byte, error) {
	if object.Size > maxSidecarMetadataBytes {
		return nil, fmt.Errorf("sidecar metadata exceeds %d bytes", maxSidecarMetadataBytes)
	}
	if provider != nil && provider.Name() == "local" {
		return readBoundedLocalFile(object.Path)
	}
	if strings.TrimSpace(object.RawURL) != "" && strings.HasPrefix(strings.TrimSpace(object.RawURL), "http") {
		return readBoundedHTTP(ctx, object.RawURL)
	}
	if provider != nil {
		link, err := provider.Link(ctx, storage.LinkRequest{Path: object.Path})
		if err == nil && strings.TrimSpace(link.URL) != "" {
			return readBoundedHTTP(ctx, link.URL)
		}
		resolved, err := provider.Get(ctx, storage.GetRequest{Path: object.Path})
		if err == nil && strings.TrimSpace(resolved.RawURL) != "" && strings.HasPrefix(strings.TrimSpace(resolved.RawURL), "http") {
			return readBoundedHTTP(ctx, resolved.RawURL)
		}
	}
	return nil, fmt.Errorf("no readable sidecar content for %s", object.Path)
}

func readBoundedLocalFile(filePath string) ([]byte, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readBounded(file)
}

func readBoundedHTTP(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(rawURL), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("sidecar metadata request failed with status %d", resp.StatusCode)
	}
	return readBounded(resp.Body)
}

func readBounded(reader io.Reader) ([]byte, error) {
	limited := io.LimitReader(reader, maxSidecarMetadataBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(content) > maxSidecarMetadataBytes {
		return nil, fmt.Errorf("sidecar metadata exceeds %d bytes", maxSidecarMetadataBytes)
	}
	return content, nil
}

func parseSidecarMetadata(ext string, content []byte) (catalogScanMetadataHints, map[string]string, error) {
	switch ext {
	case ".json":
		return parseJSONSidecarMetadata(content)
	case ".nfo":
		return parseNFOSidecarMetadata(content)
	default:
		return catalogScanMetadataHints{}, nil, fmt.Errorf("unsupported metadata sidecar extension %s", ext)
	}
}

func parseJSONSidecarMetadata(content []byte) (catalogScanMetadataHints, map[string]string, error) {
	var raw map[string]any
	if err := json.Unmarshal(content, &raw); err != nil {
		return catalogScanMetadataHints{}, nil, err
	}
	return hintsFromMap(raw), externalIDsFromMap(raw), nil
}

func parseNFOSidecarMetadata(content []byte) (catalogScanMetadataHints, map[string]string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	values := make(map[string]any)
	externalIDs := make(map[string]string)
	var current string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return parseTextNFOSidecarMetadata(content)
		}
		switch value := token.(type) {
		case xml.StartElement:
			current = strings.ToLower(strings.TrimSpace(value.Name.Local))
		case xml.CharData:
			text := strings.TrimSpace(string(value))
			if current == "" || text == "" {
				continue
			}
			if current == "uniqueid" {
				continue
			}
			if _, exists := values[current]; !exists {
				values[current] = text
			}
		case xml.EndElement:
			current = ""
		}
	}
	if value := stringFromAny(values["tmdbid"]); value != "" {
		externalIDs["tmdb"] = value
	}
	if value := stringFromAny(values["imdbid"]); value != "" {
		externalIDs["imdb"] = value
	}
	return hintsFromMap(values), externalIDs, nil
}

func parseTextNFOSidecarMetadata(content []byte) (catalogScanMetadataHints, map[string]string, error) {
	values := make(map[string]any)
	for _, line := range strings.Split(string(content), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			values[key] = value
		}
	}
	if len(values) == 0 {
		return catalogScanMetadataHints{}, nil, fmt.Errorf("nfo sidecar has no parseable metadata")
	}
	return hintsFromMap(values), externalIDsFromMap(values), nil
}

func hintsFromMap(raw map[string]any) catalogScanMetadataHints {
	season := intFromAny(firstValue(raw, "season_number", "seasonnumber", "season", "parent_index_number"))
	episode := intFromAny(firstValue(raw, "episode_number", "episodenumber", "episode", "index_number", "episodeNumber"))
	return catalogScanMetadataHints{
		Title:         stringFromAny(firstValue(raw, "title", "name")),
		OriginalTitle: stringFromAny(firstValue(raw, "original_title", "originaltitle", "originalName", "original_name")),
		Year:          intFromAny(firstValue(raw, "year", "release_year")),
		MediaType:     stringFromAny(firstValue(raw, "media_type", "type")),
		SeriesTitle:   stringFromAny(firstValue(raw, "series_title", "seriestitle", "showtitle", "show_title", "tvshowtitle")),
		SeasonNumber:  season,
		EpisodeNumber: episode,
	}
}

func externalIDsFromMap(raw map[string]any) map[string]string {
	ids := make(map[string]string)
	if nested, ok := firstValue(raw, "external_ids", "externalIds", "ids").(map[string]any); ok {
		for key, value := range nested {
			if parsed := stringFromAny(value); parsed != "" {
				ids[strings.ToLower(strings.TrimSpace(key))] = parsed
			}
		}
	}
	for _, key := range []string{"tmdb", "tmdb_id", "imdb", "imdb_id", "tvdb", "tvdb_id"} {
		if value := stringFromAny(raw[key]); value != "" {
			ids[strings.TrimSuffix(key, "_id")] = value
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func firstValue(raw map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			return value
		}
		lowerKey := strings.ToLower(key)
		for candidateKey, value := range raw {
			if strings.ToLower(candidateKey) == lowerKey {
				return value
			}
		}
	}
	return nil
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int(typed)) {
			return strconv.Itoa(int(typed))
		}
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case int:
		return strconv.Itoa(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func intFromAny(value any) *int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return &typed
		}
	case float64:
		parsed := int(typed)
		if typed == float64(parsed) && parsed > 0 {
			return &parsed
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && parsed > 0 {
			return &parsed
		}
	}
	return nil
}

func applySidecarMetadataHints(artifact catalogScanArtifact, hints catalogScanMetadataHints) catalogScanArtifact {
	if strings.TrimSpace(hints.Title) != "" {
		artifact.Title = strings.TrimSpace(hints.Title)
	}
	if strings.TrimSpace(hints.OriginalTitle) != "" {
		artifact.OriginalTitle = strings.TrimSpace(hints.OriginalTitle)
	}
	if hints.Year != nil {
		artifact.Year = hints.Year
	}
	if shouldTreatSidecarAsEpisode(hints) {
		artifact.ItemType = catalog.ItemTypeEpisode
		if strings.TrimSpace(hints.SeriesTitle) != "" {
			artifact.SeriesTitle = strings.TrimSpace(hints.SeriesTitle)
		}
		if hints.SeasonNumber != nil {
			artifact.SeasonNumber = hints.SeasonNumber
		}
		if strings.TrimSpace(artifact.SeriesTitle) != "" {
			artifact.SeriesPath = canonicalSeriesPath(artifact.SeriesTitle)
		}
		if artifact.SeriesPath != "" && artifact.SeasonNumber != nil {
			artifact.SeasonPath = fmt.Sprintf("%s/season-%02d", artifact.SeriesPath, *artifact.SeasonNumber)
		}
		if artifact.SeasonPath != "" && hints.EpisodeNumber != nil {
			artifact.EpisodeSlots = []catalogEpisodeSlot{{EpisodeNumber: *hints.EpisodeNumber, ItemPath: canonicalEpisodeItemPath(artifact.SeasonPath, *hints.EpisodeNumber)}}
		}
		return artifact
	}
	if strings.TrimSpace(hints.MediaType) == catalog.ItemTypeMovie {
		artifact.ItemType = catalog.ItemTypeMovie
	}
	return artifact
}

func shouldTreatSidecarAsEpisode(hints catalogScanMetadataHints) bool {
	mediaType := strings.ToLower(strings.TrimSpace(hints.MediaType))
	return mediaType == catalog.ItemTypeEpisode || (strings.TrimSpace(hints.SeriesTitle) != "" && hints.SeasonNumber != nil && hints.EpisodeNumber != nil)
}
