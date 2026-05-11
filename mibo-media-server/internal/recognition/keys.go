package recognition

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	CandidateTypeWork             = "work"
	CandidateTypeEpisode          = "episode"
	CandidateTypePlayableResource = "playable_resource"
	CandidateTypeVariant          = "variant"
	CandidateTypeEdition          = "edition"
	CandidateTypeSupplemental     = "supplemental"
	CandidateTypeDuplicateBinary  = "duplicate_binary"

	WorkKindMovie   = "movie"
	WorkKindSeries  = "series"
	WorkKindSeason  = "season"
	WorkKindEpisode = "episode"

	ResourceKindSingleFile   = "single_file"
	ResourceKindMultiPart    = "multi_part"
	ResourceKindMultiEpisode = "multi_episode"
)

var keyUnsafePattern = regexp.MustCompile(`[^a-z0-9]+`)

type Scope struct {
	StorageProvider string
	RootPath        string
	ScopePath       string
}

type MovieWorkInput struct {
	Title string
	Year  *int
}

type EpisodeInput struct {
	SeriesTitle   string
	SeasonNumber  int
	EpisodeNumber int
}

type ResourceInput struct {
	StorageProvider   string
	StoragePath       string
	StableIdentityKey string
}

type VariantInput struct {
	Quality      string
	SourceTags   []string
	Codec        string
	Audio        string
	Subtitle     string
	HDR          string
	Container    string
	ReleaseGroup string
}

func ManifestKey(scope Scope, classifierVersion string) string {
	return joinKey("manifest", scope.StorageProvider, cleanPathKey(scope.RootPath), cleanPathKey(firstNonEmpty(scope.ScopePath, scope.RootPath)), classifierVersion)
}

func MovieWorkKey(input MovieWorkInput) string {
	base := normalizeTitleKey(input.Title)
	if base == "" {
		return ""
	}
	if input.Year != nil && *input.Year > 0 {
		return joinKey(CandidateTypeWork, WorkKindMovie, base, strconv.Itoa(*input.Year))
	}
	return joinKey(CandidateTypeWork, WorkKindMovie, base)
}

func SeriesWorkKey(title string) string {
	base := normalizeTitleKey(title)
	if base == "" {
		return ""
	}
	return joinKey(CandidateTypeWork, WorkKindSeries, base)
}

func SeasonWorkKey(seriesTitle string, seasonNumber int) string {
	series := SeriesWorkKey(seriesTitle)
	if series == "" || seasonNumber <= 0 {
		return ""
	}
	return joinKey(CandidateTypeWork, WorkKindSeason, series, fmt.Sprintf("s%02d", seasonNumber))
}

func EpisodeKey(input EpisodeInput) string {
	season := SeasonWorkKey(input.SeriesTitle, input.SeasonNumber)
	if season == "" || input.EpisodeNumber <= 0 {
		return ""
	}
	return joinKey(CandidateTypeEpisode, season, fmt.Sprintf("e%02d", input.EpisodeNumber))
}

func PlayableResourceKey(input ResourceInput) string {
	provider := strings.TrimSpace(input.StorageProvider)
	if provider == "" {
		provider = "local"
	}
	if stable := strings.TrimSpace(input.StableIdentityKey); stable != "" {
		return joinKey(CandidateTypePlayableResource, provider, "stable", stable)
	}
	storagePath := cleanPathKey(input.StoragePath)
	if storagePath == "" {
		return ""
	}
	return joinKey(CandidateTypePlayableResource, provider, "path", storagePath)
}

func VariantKey(input VariantInput) string {
	parts := []string{input.Quality, input.Codec, input.Audio, input.Subtitle, input.HDR, input.Container, input.ReleaseGroup}
	parts = append(parts, input.SourceTags...)
	return keyedTrait(CandidateTypeVariant, parts)
}

func EditionKey(edition string) string {
	return keyedTrait(CandidateTypeEdition, []string{edition})
}

func SupplementalKey(parentKey string, role string, resourceKey string) string {
	return joinKey(CandidateTypeSupplemental, parentKey, normalizeToken(role), resourceKey)
}

func DuplicateBinaryKey(hashKind string, hashValue string) string {
	return joinKey(CandidateTypeDuplicateBinary, normalizeToken(hashKind), strings.ToLower(strings.TrimSpace(hashValue)))
}

func keyedTrait(prefix string, values []string) string {
	traits := make([]string, 0, len(values))
	for _, value := range values {
		if token := normalizeToken(value); token != "" {
			traits = append(traits, token)
		}
	}
	sort.Strings(traits)
	if len(traits) == 0 {
		return ""
	}
	return joinKey(append([]string{prefix}, traits...)...)
}

func joinKey(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return strings.Join(cleaned, ":")
}

func normalizeTitleKey(value string) string {
	return normalizeToken(value)
}

func normalizeToken(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	lower = strings.ReplaceAll(lower, "&", " and ")
	lower = keyUnsafePattern.ReplaceAllString(lower, "-")
	return strings.Trim(lower, "-")
}

func cleanPathKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return path.Clean(trimmed)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
