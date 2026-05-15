package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/scanrecognition"
	"github.com/atlan/mibo-media-server/internal/storage"
	"github.com/atlan/mibo-media-server/internal/titleclean"
	"gorm.io/gorm"
)

type contentShapeFingerprintInput struct {
	LibraryID         uint
	StorageProvider   string
	RootPath          string
	DirectoryPath     string
	ClassifierVersion string
	ScanPolicy        database.LibraryScanPolicy
	ExclusionRules    []database.ScanExclusionRule
	Snapshot          scanDirectorySnapshot
	VisibleVideoCount int
}

type contentShapeFingerprintSummary struct {
	VisibleVideoCount int
}

type contentShapeDirectoryProfile struct {
	LibraryType          string
	Path                 string
	RootPath             string
	VideoCount           int
	NonExtraVideoCount   int
	AttachmentCount      int
	ExplicitEpisodeCount int
	LeadingNumericCount  int
	NumericSequence      []int
	SequenceCoverage     float64
	SequenceGaps         []int
	YearDensity          float64
	TitleUniqueness      float64
	TitleYearCount       int
	TitleUniqueCount     int
	CommonTitleStem      string
	VersionEvidenceCount int
	SeasonHint           *int
	SidecarHintCount     int
	CategoryPathHints    []string
}

func loadOrBuildContentShapeProfile(ctx context.Context, db *gorm.DB, scope contentShapeScope, snapshot scanDirectorySnapshot, scanPolicy database.LibraryScanPolicy, exclusionRules []database.ScanExclusionRule, tokenCache *filenameTokenProfileCache) (database.ContentShapeProfile, bool, error) {
	profile, _, reused, err := loadOrBuildContentShapeProfileWithBuilt(ctx, db, scope, snapshot, scanPolicy, exclusionRules, tokenCache)
	return profile, reused, err
}

func loadOrBuildContentShapeProfileWithBuilt(ctx context.Context, db *gorm.DB, scope contentShapeScope, snapshot scanDirectorySnapshot, scanPolicy database.LibraryScanPolicy, exclusionRules []database.ScanExclusionRule, tokenCache *filenameTokenProfileCache) (database.ContentShapeProfile, contentShapeDirectoryProfile, bool, error) {
	summary := summarizeContentShapeFingerprintSnapshot(snapshot)
	scope.Fingerprint = contentShapeDirectoryFingerprint(contentShapeFingerprintInput{LibraryID: scope.LibraryID, StorageProvider: scope.StorageProvider, RootPath: scope.RootPath, DirectoryPath: scope.DirectoryPath, ClassifierVersion: scope.ClassifierVersion, ScanPolicy: scanPolicy, ExclusionRules: exclusionRules, Snapshot: snapshot, VisibleVideoCount: summary.VisibleVideoCount})
	if profile, reused, err := loadReusableContentShapeProfile(ctx, db, scope); err != nil || reused {
		return profile, contentShapeDirectoryProfile{}, reused, err
	}
	built := buildContentShapeDirectoryProfile("auto", scope.RootPath, snapshot, tokenCache)
	profile := contentShapeDatabaseProfile(scope, built)
	if err := saveContentShapeProfile(ctx, db, &profile); err != nil {
		return database.ContentShapeProfile{}, built, false, err
	}
	profile, _, err := loadReusableContentShapeProfile(ctx, db, scope)
	return profile, built, false, err
}

func summarizeContentShapeFingerprintSnapshot(snapshot scanDirectorySnapshot) contentShapeFingerprintSummary {
	summary := contentShapeFingerprintSummary{}
	for _, object := range snapshot.Objects {
		if object.IsDir || !isVideoFile(object.Path) {
			continue
		}
		summary.VisibleVideoCount++
	}
	return summary
}

func contentShapeDatabaseProfile(scope contentShapeScope, profile contentShapeDirectoryProfile) database.ContentShapeProfile {
	now := time.Now().UTC()
	evidence := map[string]any{
		"sequence_gaps":       profile.SequenceGaps,
		"numeric_sequence":    profile.NumericSequence,
		"category_path_hints": profile.CategoryPathHints,
	}
	evidenceJSON, _ := json.Marshal(evidence)
	confidence := profile.SequenceCoverage
	return database.ContentShapeProfile{
		LibraryID:            scope.LibraryID,
		MediaSourceID:        scope.MediaSourceID,
		LibraryPathID:        scope.LibraryPathID,
		StorageProvider:      strings.TrimSpace(scope.StorageProvider),
		RootPath:             strings.TrimSpace(scope.RootPath),
		DirectoryPath:        strings.TrimSpace(scope.DirectoryPath),
		ClassifierVersion:    strings.TrimSpace(scope.ClassifierVersion),
		Fingerprint:          strings.TrimSpace(scope.Fingerprint),
		VideoCount:           profile.VideoCount,
		NonExtraVideoCount:   profile.NonExtraVideoCount,
		AttachmentCount:      profile.AttachmentCount,
		ExplicitEpisodeCount: profile.ExplicitEpisodeCount,
		LeadingNumericCount:  profile.LeadingNumericCount,
		SequenceCoverage:     &profile.SequenceCoverage,
		YearDensity:          &profile.YearDensity,
		TitleUniqueness:      &profile.TitleUniqueness,
		CommonTitleStem:      profile.CommonTitleStem,
		SeasonHint:           contentShapeSeasonHint(profile.SeasonHint),
		SidecarHintsJSON:     mustJSON(profile.SidecarHintCount),
		Confidence:           &confidence,
		ReviewState:          "auto",
		EvidenceJSON:         string(evidenceJSON),
		LastObservedAt:       now,
	}
}

func contentShapeSeasonHint(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("season-%02d", *value)
}

func buildContentShapeDirectoryProfile(libraryType string, libraryRoot string, snapshot scanDirectorySnapshot, tokenCache *filenameTokenProfileCache) contentShapeDirectoryProfile {
	profile := contentShapeDirectoryProfile{LibraryType: libraryType, Path: snapshot.Path, RootPath: libraryRoot, SeasonHint: scanrecognition.ParseFolderName(path.Base(snapshot.Path)).Season}
	titleCounts := make(map[string]int)
	categoryHints := make(map[string]struct{})
	for _, object := range snapshot.Objects {
		if object.IsDir {
			if hint := contentShapeCategoryHint(object.Path); hint != "" {
				categoryHints[hint] = struct{}{}
			}
			continue
		}
		if isSidecarLikeObject(object) {
			profile.SidecarHintCount++
		}
		if !isVideoFile(object.Path) {
			continue
		}
		profile.VideoCount++
		model := filenameTokenProfileForPath(tokenCache, object.Path)
		if model.VideoSignal.IsExtra {
			profile.AttachmentCount++
			continue
		}
		profile.NonExtraVideoCount++
		if model.VideoSignal.EpisodeSource == "explicit" && (model.VideoSignal.Episode != nil || len(model.VideoSignal.EpisodeNumbers) > 0) {
			profile.ExplicitEpisodeCount++
		}
		if model.VideoSignal.LeadingNumber != nil {
			profile.LeadingNumericCount++
		}
		if model.VideoSignal.Episode != nil {
			profile.NumericSequence = append(profile.NumericSequence, *model.VideoSignal.Episode)
		}
		if filenameSignalYear(model) != nil && model.VideoSignal.Episode == nil && len(model.VideoSignal.EpisodeNumbers) == 0 {
			profile.TitleYearCount++
		}
		stem := normalizeVersionCompareTitle(filenameSignalTitleCandidate(model))
		if stem != "" {
			titleCounts[stem]++
		}
		if strings.TrimSpace(model.VideoSignal.Quality) != "" || strings.TrimSpace(model.VideoSignal.Edition) != "" || strings.TrimSpace(model.VideoSignal.ReleaseGroup) != "" {
			profile.VersionEvidenceCount++
		}
	}
	sort.Ints(profile.NumericSequence)
	maxTitleCount := 0
	for stem, count := range titleCounts {
		if count > maxTitleCount || (count == maxTitleCount && stem < profile.CommonTitleStem) {
			profile.CommonTitleStem = stem
			maxTitleCount = count
		}
	}
	profile.TitleUniqueCount = len(titleCounts)
	profile.SequenceCoverage, profile.SequenceGaps = contentShapeSequenceCoverage(profile.NumericSequence, profile.NonExtraVideoCount)
	if profile.NonExtraVideoCount > 0 {
		profile.YearDensity = float64(profile.TitleYearCount) / float64(profile.NonExtraVideoCount)
		profile.TitleUniqueness = float64(profile.TitleUniqueCount) / float64(profile.NonExtraVideoCount)
	}
	profile.CategoryPathHints = sortedContentShapeHints(categoryHints)
	return profile
}

func contentShapeSequenceCoverage(sequence []int, nonExtraVideoCount int) (float64, []int) {
	if nonExtraVideoCount == 0 || len(sequence) == 0 {
		return 0, nil
	}
	unique := make(map[int]struct{}, len(sequence))
	for _, value := range sequence {
		if value > 0 {
			unique[value] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return 0, nil
	}
	numbers := make([]int, 0, len(unique))
	for value := range unique {
		numbers = append(numbers, value)
	}
	sort.Ints(numbers)
	gaps := make([]int, 0)
	for expected := numbers[0]; expected <= numbers[len(numbers)-1]; expected++ {
		if _, ok := unique[expected]; !ok {
			gaps = append(gaps, expected)
		}
	}
	return float64(len(unique)) / float64(nonExtraVideoCount), gaps
}

func isSidecarLikeObject(object storage.Object) bool {
	ext := sidecarExtension(object.Path)
	return ext != "" || isCatalogScanArtworkFile(object.Path)
}

func contentShapeCategoryHint(objectPath string) string {
	segment := strings.ToLower(strings.TrimSpace(path.Base(objectPath)))
	switch segment {
	case "season", "seasons", "extras", "trailers", "samples", "specials", "movies", "movie", "show", "shows", "tv":
		return segment
	}
	if scanrecognition.ParseFolderName(segment).Season != nil {
		return "season"
	}
	return ""
}

func sortedContentShapeHints(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func normalizeVersionCompareTitle(input string) string {
	return titleclean.NormalizeMovieWorkTitle(input)
}

func contentShapeDirectoryFingerprint(input contentShapeFingerprintInput) string {
	parts := []string{
		fmt.Sprintf("library=%d", input.LibraryID),
		"provider=" + strings.TrimSpace(input.StorageProvider),
		"root=" + strings.TrimSpace(input.RootPath),
		"dir=" + strings.TrimSpace(input.DirectoryPath),
		"classifier=" + strings.TrimSpace(input.ClassifierVersion),
		contentShapeScanPolicyFingerprint(input.ScanPolicy),
		contentShapeExclusionFingerprint(input.ExclusionRules),
		fmt.Sprintf("visible_videos=%d", input.VisibleVideoCount),
	}
	childParts := make([]string, 0, len(input.Snapshot.Objects))
	for _, object := range input.Snapshot.Objects {
		modified := ""
		if object.Modified != nil {
			modified = object.Modified.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
		}
		childParts = append(childParts, strings.Join([]string{
			strings.TrimSpace(path.Base(object.Path)),
			fmt.Sprintf("dir=%t", object.IsDir),
			fmt.Sprintf("size=%d", object.Size),
			"modified=" + modified,
			"stable=" + strings.TrimSpace(object.StableIdentity),
		}, "|"))
	}
	sort.Strings(childParts)
	parts = append(parts, childParts...)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func contentShapeScanPolicyFingerprint(policy database.LibraryScanPolicy) string {
	return strings.Join([]string{
		fmt.Sprintf("ignore_hidden=%t", policy.IgnoreHiddenFiles),
		"ignore_ext=" + strings.TrimSpace(policy.IgnoreFileExtensionsJSON),
		fmt.Sprintf("min_size=%d", policy.MinFileSizeBytes),
		fmt.Sprintf("sample_size=%d", policy.SampleIgnoreSizeBytes),
		fmt.Sprintf("rules=%t", policy.ConfigurableExclusionRules),
	}, "|")
}

func contentShapeExclusionFingerprint(rules []database.ScanExclusionRule) string {
	parts := make([]string, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		parts = append(parts, strings.Join([]string{rule.Key, rule.RuleType, rule.Value, rule.Reason}, "|"))
	}
	sort.Strings(parts)
	return "exclusions=" + strings.Join(parts, ";")
}
