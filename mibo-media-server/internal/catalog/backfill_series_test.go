package catalog

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestLegacyBackfillSeries(t *testing.T) {
	svc, ctx := newTestService(t)

	libraryID := uint(11)
	run, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID}, 77)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	confidence := 0.91
	year := 2025
	runtimeSeconds := 2700
	seasonNumber := 1
	episodeOneNumber := 1
	episodeTwoNumber := 2
	releaseDateOne := "2025-01-08"
	releaseDateTwo := "2025-01-15"
	episodeOne := database.MediaItem{
		LibraryID:          libraryID,
		Type:               "episode",
		Title:              "Pilot",
		SeriesTitle:        "Provider Show",
		Overview:           "Series overview",
		PosterURL:          "https://images.example.com/provider-show/poster.jpg",
		BackdropURL:        "https://images.example.com/provider-show/backdrop.jpg",
		LogoURL:            "https://images.example.com/provider-show/logo.png",
		Year:               &year,
		ReleaseDate:        releaseDateOne,
		RuntimeSeconds:     &runtimeSeconds,
		SeasonNumber:       &seasonNumber,
		EpisodeNumber:      &episodeOneNumber,
		SourcePath:         "/library/shows/provider-show/Season 01/pilot.mkv",
		MatchStatus:        "matched",
		MetadataProvider:   "tmdb",
		ExternalID:         "tv:777",
		MetadataConfidence: &confidence,
		Status:             "ready",
	}
	episodeTwo := database.MediaItem{
		LibraryID:          libraryID,
		Type:               "episode",
		Title:              "Second Wave",
		SeriesTitle:        "Provider Show",
		Overview:           "Series overview",
		Year:               &year,
		ReleaseDate:        releaseDateTwo,
		RuntimeSeconds:     &runtimeSeconds,
		SeasonNumber:       &seasonNumber,
		EpisodeNumber:      &episodeTwoNumber,
		SourcePath:         "/library/shows/provider-show/Season 01/second-wave.mkv",
		MatchStatus:        "matched",
		MetadataProvider:   "tmdb",
		ExternalID:         "tv:777",
		MetadataConfidence: &confidence,
		Status:             "ready",
	}
	for _, legacyEpisode := range []*database.MediaItem{&episodeOne, &episodeTwo} {
		if err := svc.db.WithContext(ctx).Create(legacyEpisode).Error; err != nil {
			t.Fatalf("create legacy episode %q: %v", legacyEpisode.Title, err)
		}
	}

	modifiedAt := time.Date(2026, time.April, 25, 8, 0, 0, 0, time.UTC)
	durationSeconds := 2700.0
	legacyFiles := []database.MediaFile{
		{
			LibraryID:          libraryID,
			MediaItemID:        &episodeOne.ID,
			StoragePath:        episodeOne.SourcePath,
			StableIdentityKey:  "stable:provider-show:s01e01",
			IdentitySource:     "scan",
			IdentityStatus:     "confirmed",
			ProviderName:       "local",
			ProviderHashesJSON: `{"sha256":"ep1"}`,
			ReviewStatus:       "none",
			Container:          "mkv",
			SizeBytes:          1001,
			ProbeStatus:        "complete",
			DurationSeconds:    &durationSeconds,
			LastModifiedAt:     &modifiedAt,
		},
		{
			LibraryID:          libraryID,
			MediaItemID:        &episodeTwo.ID,
			StoragePath:        episodeTwo.SourcePath,
			StableIdentityKey:  "stable:provider-show:s01e02",
			IdentitySource:     "scan",
			IdentityStatus:     "confirmed",
			ProviderName:       "local",
			ProviderHashesJSON: `{"sha256":"ep2"}`,
			ReviewStatus:       "none",
			Container:          "mkv",
			SizeBytes:          1002,
			ProbeStatus:        "complete",
			DurationSeconds:    &durationSeconds,
			LastModifiedAt:     &modifiedAt,
		},
	}
	for _, legacyFile := range legacyFiles {
		file := legacyFile
		if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create legacy file %q: %v", legacyFile.StoragePath, err)
		}
	}

	if err := svc.backfillSeries(ctx, run); err != nil {
		t.Fatalf("backfill series: %v", err)
	}

	finalized, err := svc.finalizeLegacyBackfillRun(ctx, run.ID, LegacyBackfillStatusCompleted, "")
	if err != nil {
		t.Fatalf("finalize run: %v", err)
	}
	if finalized.SuccessCount != 2 || finalized.ConflictCount != 0 || finalized.OrphanFileCount != 0 || finalized.DuplicateEpisodeCandidateCount != 0 {
		t.Fatalf("unexpected finalized counts: %#v", finalized)
	}

	var seriesItems []database.CatalogItem
	if err := svc.db.WithContext(ctx).Where("library_id = ? AND type = ?", libraryID, ItemTypeSeries).Order("id asc").Find(&seriesItems).Error; err != nil {
		t.Fatalf("list series items: %v", err)
	}
	if len(seriesItems) != 1 {
		t.Fatalf("expected one series item, got %#v", seriesItems)
	}
	series := seriesItems[0]
	if series.Title != episodeOne.SeriesTitle {
		t.Fatalf("expected series title %q, got %q", episodeOne.SeriesTitle, series.Title)
	}
	if series.AvailabilityStatus != AvailabilityAvailable || series.GovernanceStatus != GovernanceMatched {
		t.Fatalf("unexpected series state: %#v", series)
	}

	seasons, err := svc.ListChildren(ctx, series.ID)
	if err != nil {
		t.Fatalf("list series children: %v", err)
	}
	if len(seasons) != 1 {
		t.Fatalf("expected one season, got %#v", seasons)
	}
	season := seasons[0]
	if season.Type != ItemTypeSeason || season.IndexNumber == nil || *season.IndexNumber != seasonNumber {
		t.Fatalf("unexpected season row: %#v", season)
	}
	if season.AvailabilityStatus != AvailabilityAvailable {
		t.Fatalf("expected available season, got %#v", season)
	}

	episodes, err := svc.ListChildren(ctx, season.ID)
	if err != nil {
		t.Fatalf("list season children: %v", err)
	}
	if len(episodes) != 2 {
		t.Fatalf("expected two canonical episodes, got %#v", episodes)
	}
	if episodes[0].IndexNumber == nil || episodes[1].IndexNumber == nil || *episodes[0].IndexNumber != episodeOneNumber || *episodes[1].IndexNumber != episodeTwoNumber {
		t.Fatalf("unexpected episode numbering: %#v", episodes)
	}
	if episodes[0].AvailabilityStatus != AvailabilityAvailable || episodes[1].AvailabilityStatus != AvailabilityAvailable {
		t.Fatalf("expected available episodes, got %#v", episodes)
	}

	var externalIDs []database.CatalogExternalID
	if err := svc.db.WithContext(ctx).Order("id asc").Find(&externalIDs).Error; err != nil {
		t.Fatalf("list external ids: %v", err)
	}
	if len(externalIDs) != 1 {
		t.Fatalf("expected one series external id, got %#v", externalIDs)
	}
	if externalIDs[0].ItemID != series.ID || externalIDs[0].Provider != episodeOne.MetadataProvider || externalIDs[0].ProviderType != "series" || externalIDs[0].ExternalID != episodeOne.ExternalID {
		t.Fatalf("unexpected series external id: %#v", externalIDs[0])
	}

	var sources []database.MetadataSource
	if err := svc.db.WithContext(ctx).Order("id asc").Find(&sources).Error; err != nil {
		t.Fatalf("list metadata sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected one metadata source, got %#v", sources)
	}
	if sources[0].ItemID != series.ID || sources[0].SourceType != SourceTypeProvider || sources[0].SourceName != episodeOne.MetadataProvider {
		t.Fatalf("unexpected series metadata source: %#v", sources[0])
	}
	var sourcePayload map[string]any
	if err := json.Unmarshal([]byte(sources[0].PayloadJSON), &sourcePayload); err != nil {
		t.Fatalf("unmarshal series metadata payload: %v", err)
	}
	if sourcePayload["legacy_media_item_id"] != float64(episodeOne.ID) || sourcePayload["match_status"] != episodeOne.MatchStatus {
		t.Fatalf("unexpected series metadata payload: %#v", sourcePayload)
	}

	var images []database.ItemImage
	if err := svc.db.WithContext(ctx).Where("item_id = ?", series.ID).Order("image_type asc").Find(&images).Error; err != nil {
		t.Fatalf("list series images: %v", err)
	}
	if len(images) != 3 {
		t.Fatalf("expected three selected series images, got %#v", images)
	}

	assertTableRowCount(t, ctx, svc.db, &database.InventoryFile{}, 2)
	assertTableRowCount(t, ctx, svc.db, &database.MediaAsset{}, 2)
	assertTableRowCount(t, ctx, svc.db, &database.AssetFile{}, 2)
	assertTableRowCount(t, ctx, svc.db, &database.AssetItem{}, 2)

	var assetItems []database.AssetItem
	if err := svc.db.WithContext(ctx).Order("id asc").Find(&assetItems).Error; err != nil {
		t.Fatalf("list asset items: %v", err)
	}
	for _, assetItem := range assetItems {
		if assetItem.ItemID != episodes[assetItem.SegmentIndex].ID && assetItem.ItemID != episodes[0].ID && assetItem.ItemID != episodes[1].ID {
			t.Fatalf("unexpected asset item link: %#v", assetItem)
		}
		if assetItem.Role != "primary" || assetItem.Source != legacyBackfillSource {
			t.Fatalf("unexpected asset item values: %#v", assetItem)
		}
	}

	report, err := svc.GetLegacyBackfillRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("load backfill report: %v", err)
	}
	if len(report.Entries) != 2 {
		t.Fatalf("expected one success entry per playable legacy episode, got %#v", report.Entries)
	}
	for _, entry := range report.Entries {
		if entry.EntryType != LegacyBackfillEntryTypeSuccess {
			t.Fatalf("expected success-only entries, got %#v", report.Entries)
		}
		if entry.CatalogItemID == nil || (*entry.CatalogItemID != episodes[0].ID && *entry.CatalogItemID != episodes[1].ID) {
			t.Fatalf("unexpected canonical episode entry link: %#v", entry)
		}
	}
}

func TestLegacyBackfillSeriesConflicts(t *testing.T) {
	svc, ctx := newTestService(t)

	libraryID := uint(12)
	run, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID}, 78)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	confidence := 0.82
	year := 2024
	seasonOne := 1
	episodeOne := 1
	episodeThree := 3
	duplicateA := database.MediaItem{
		LibraryID:          libraryID,
		Type:               "episode",
		Title:              "Fallback Pilot",
		SeriesTitle:        "Fallback Show",
		Year:               &year,
		SeasonNumber:       &seasonOne,
		EpisodeNumber:      &episodeOne,
		SourcePath:         "/library/shows/fallback-show/Season 01/fallback-pilot-a.mkv",
		MatchStatus:        "matched",
		MetadataProvider:   "tmdb",
		ExternalID:         "tv:880",
		MetadataConfidence: &confidence,
		Status:             "ready",
	}
	duplicateB := database.MediaItem{
		LibraryID:     libraryID,
		Type:          "episode",
		Title:         "Fallback Pilot Alt",
		SeriesTitle:   "Fallback Show",
		Year:          &year,
		SeasonNumber:  &seasonOne,
		EpisodeNumber: &episodeOne,
		SourcePath:    "/library/shows/fallback-show/Season 01/fallback-pilot-b.mkv",
		MatchStatus:   "matched",
		Status:        "ready",
	}
	fallbackTitleOnly := database.MediaItem{
		LibraryID:     libraryID,
		Type:          "episode",
		Title:         "Title Fallback Episode",
		SeriesTitle:   "Title Fallback Show",
		Year:          &year,
		SeasonNumber:  &seasonOne,
		EpisodeNumber: &episodeThree,
		SourcePath:    "/library/shows/title-fallback/Season 01/title-fallback-episode.mkv",
		MatchStatus:   "needs_review",
		Status:        "ready",
	}
	missingIdentity := database.MediaItem{
		LibraryID:     libraryID,
		Type:          "episode",
		Title:         "Identity Missing Episode",
		SeasonNumber:  &seasonOne,
		EpisodeNumber: &episodeThree,
		SourcePath:    "/library/shows/unknown/Season 01/identity-missing.mkv",
		MatchStatus:   "pending",
		Status:        "ready",
	}
	for _, legacyEpisode := range []*database.MediaItem{&duplicateB, &duplicateA, &fallbackTitleOnly, &missingIdentity} {
		if err := svc.db.WithContext(ctx).Create(legacyEpisode).Error; err != nil {
			t.Fatalf("create legacy episode %q: %v", legacyEpisode.Title, err)
		}
	}

	durationSeconds := 1800.0
	modifiedAt := time.Date(2026, time.April, 25, 9, 0, 0, 0, time.UTC)
	legacyFiles := []database.MediaFile{
		{
			LibraryID:          libraryID,
			MediaItemID:        &duplicateA.ID,
			StoragePath:        duplicateA.SourcePath,
			StableIdentityKey:  "stable:fallback-show:a",
			IdentitySource:     "scan",
			IdentityStatus:     "confirmed",
			ProviderName:       "local",
			ProviderHashesJSON: `{"sha256":"dup-a"}`,
			ReviewStatus:       "none",
			Container:          "mkv",
			SizeBytes:          2001,
			ProbeStatus:        "complete",
			DurationSeconds:    &durationSeconds,
			LastModifiedAt:     &modifiedAt,
		},
		{
			LibraryID:          libraryID,
			MediaItemID:        &duplicateB.ID,
			StoragePath:        duplicateB.SourcePath,
			StableIdentityKey:  "stable:fallback-show:b",
			IdentitySource:     "scan",
			IdentityStatus:     "confirmed",
			ProviderName:       "local",
			ProviderHashesJSON: `{"sha256":"dup-b"}`,
			ReviewStatus:       "none",
			Container:          "mkv",
			SizeBytes:          2002,
			ProbeStatus:        "complete",
			DurationSeconds:    &durationSeconds,
			LastModifiedAt:     &modifiedAt,
		},
		{
			LibraryID:          libraryID,
			MediaItemID:        &fallbackTitleOnly.ID,
			StoragePath:        fallbackTitleOnly.SourcePath,
			StableIdentityKey:  "stable:title-fallback",
			IdentitySource:     "scan",
			IdentityStatus:     "confirmed",
			ProviderName:       "local",
			ProviderHashesJSON: `{"sha256":"fallback"}`,
			ReviewStatus:       "none",
			Container:          "mkv",
			SizeBytes:          2003,
			ProbeStatus:        "complete",
			DurationSeconds:    &durationSeconds,
			LastModifiedAt:     &modifiedAt,
		},
		{
			LibraryID:          libraryID,
			StoragePath:        "/library/shows/orphans/orphan-file.mkv",
			StableIdentityKey:  "stable:orphan-file",
			IdentitySource:     "scan",
			IdentityStatus:     "confirmed",
			ProviderName:       "local",
			ProviderHashesJSON: `{"sha256":"orphan"}`,
			ReviewStatus:       "none",
			Container:          "mkv",
			SizeBytes:          2004,
			ProbeStatus:        "complete",
			DurationSeconds:    &durationSeconds,
			LastModifiedAt:     &modifiedAt,
		},
	}
	for _, legacyFile := range legacyFiles {
		file := legacyFile
		if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create legacy file %q: %v", legacyFile.StoragePath, err)
		}
	}

	if err := svc.backfillSeries(ctx, run); err != nil {
		t.Fatalf("backfill series: %v", err)
	}

	finalized, err := svc.finalizeLegacyBackfillRun(ctx, run.ID, LegacyBackfillStatusCompleted, "")
	if err != nil {
		t.Fatalf("finalize run: %v", err)
	}
	if finalized.SuccessCount != 3 || finalized.ConflictCount != 1 || finalized.OrphanFileCount != 1 || finalized.DuplicateEpisodeCandidateCount != 1 {
		t.Fatalf("unexpected finalized counts: %#v", finalized)
	}

	var seriesItems []database.CatalogItem
	if err := svc.db.WithContext(ctx).Where("library_id = ? AND type = ?", libraryID, ItemTypeSeries).Order("id asc").Find(&seriesItems).Error; err != nil {
		t.Fatalf("list series items: %v", err)
	}
	if len(seriesItems) != 2 {
		t.Fatalf("expected provider-backed and title-fallback series rows, got %#v", seriesItems)
	}

	var duplicateEpisode database.CatalogItem
	if err := svc.db.WithContext(ctx).
		Where("library_id = ? AND type = ? AND title = ?", libraryID, ItemTypeEpisode, duplicateA.Title).
		First(&duplicateEpisode).Error; err != nil {
		t.Fatalf("load canonical duplicate episode: %v", err)
	}
	assertTableRowCount(t, ctx, svc.db, &database.InventoryFile{}, 3)
	assertTableRowCount(t, ctx, svc.db, &database.MediaAsset{}, 3)

	var duplicateAssetItems []database.AssetItem
	if err := svc.db.WithContext(ctx).Where("item_id = ?", duplicateEpisode.ID).Order("id asc").Find(&duplicateAssetItems).Error; err != nil {
		t.Fatalf("list duplicate episode asset links: %v", err)
	}
	if len(duplicateAssetItems) != 2 {
		t.Fatalf("expected both playable duplicate candidates to link to the same canonical episode, got %#v", duplicateAssetItems)
	}
	for _, assetItem := range duplicateAssetItems {
		if assetItem.Source != legacyBackfillSource {
			t.Fatalf("expected duplicate candidate asset link source %q, got %#v", legacyBackfillSource, assetItem)
		}
	}

	report, err := svc.GetLegacyBackfillRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("load backfill report: %v", err)
	}
	if len(report.Entries) != 6 {
		t.Fatalf("expected success, duplicate candidate, conflict, and orphan entries, got %#v", report.Entries)
	}
	entryCounts := map[string]int{}
	for _, entry := range report.Entries {
		entryCounts[entry.EntryType]++
	}
	if entryCounts[LegacyBackfillEntryTypeSuccess] != 3 {
		t.Fatalf("expected three success entries, got %#v", entryCounts)
	}
	if entryCounts[LegacyBackfillEntryTypeDuplicateEpisodeCandidate] != 1 {
		t.Fatalf("expected duplicate episode candidate entry, got %#v", entryCounts)
	}
	if entryCounts[LegacyBackfillEntryTypeConflict] != 1 {
		t.Fatalf("expected conflict entry, got %#v", entryCounts)
	}
	if entryCounts[LegacyBackfillEntryTypeOrphanFile] != 1 {
		t.Fatalf("expected orphan_file entry, got %#v", entryCounts)
	}

	var duplicateEntry, orphanEntry, conflictEntry *LegacyBackfillEntry
	for idx := range report.Entries {
		entry := &report.Entries[idx]
		switch entry.EntryType {
		case LegacyBackfillEntryTypeDuplicateEpisodeCandidate:
			duplicateEntry = entry
		case LegacyBackfillEntryTypeOrphanFile:
			orphanEntry = entry
		case LegacyBackfillEntryTypeConflict:
			conflictEntry = entry
		}
	}
	if duplicateEntry == nil || duplicateEntry.LegacyMediaItemID == nil || *duplicateEntry.LegacyMediaItemID != duplicateB.ID {
		t.Fatalf("expected duplicate_episode_candidate entry for second slot claimant, got %#v", duplicateEntry)
	}
	if orphanEntry == nil || orphanEntry.LegacyMediaFileID == nil || orphanEntry.StoragePath != "/library/shows/orphans/orphan-file.mkv" {
		t.Fatalf("expected orphan_file entry for orphaned media file, got %#v", orphanEntry)
	}
	if conflictEntry == nil || conflictEntry.LegacyMediaItemID == nil || *conflictEntry.LegacyMediaItemID != missingIdentity.ID {
		t.Fatalf("expected conflict entry for missing identity row, got %#v", conflictEntry)
	}
	if conflictEntry.Message == "" {
		t.Fatalf("expected conflict message to explain missing identity, got %#v", conflictEntry)
	}
}

func TestLegacyBackfillSeriesSeparatesDistinctProviderBackedShows(t *testing.T) {
	svc, ctx := newTestService(t)

	libraryID := uint(13)
	run, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID}, 79)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	year := 2023
	seasonNumber := 1
	episodeNumber := 1
	runtimeSeconds := 1800
	first := database.MediaItem{
		LibraryID:        libraryID,
		Type:             ItemTypeEpisode,
		Title:            "Pilot",
		SeriesTitle:      "Shared Title",
		Year:             &year,
		SeasonNumber:     &seasonNumber,
		EpisodeNumber:    &episodeNumber,
		SourcePath:       "/library/shows/shared-title-a/Season 01/pilot.mkv",
		MatchStatus:      "matched",
		MetadataProvider: "tmdb",
		ExternalID:       "tv:100",
		RuntimeSeconds:   &runtimeSeconds,
		Status:           "ready",
	}
	second := database.MediaItem{
		LibraryID:        libraryID,
		Type:             ItemTypeEpisode,
		Title:            "Pilot",
		SeriesTitle:      "Shared Title",
		Year:             &year,
		SeasonNumber:     &seasonNumber,
		EpisodeNumber:    &episodeNumber,
		SourcePath:       "/library/shows/shared-title-b/Season 01/pilot.mkv",
		MatchStatus:      "matched",
		MetadataProvider: "tmdb",
		ExternalID:       "tv:200",
		RuntimeSeconds:   &runtimeSeconds,
		Status:           "ready",
	}
	for _, item := range []*database.MediaItem{&first, &second} {
		if err := svc.db.WithContext(ctx).Create(item).Error; err != nil {
			t.Fatalf("create legacy episode %q: %v", item.SourcePath, err)
		}
	}

	durationSeconds := 1800.0
	for _, file := range []database.MediaFile{
		{LibraryID: libraryID, MediaItemID: &first.ID, StoragePath: first.SourcePath, StableIdentityKey: "stable:shared-a", ProviderName: "local", ProbeStatus: "complete", DurationSeconds: &durationSeconds},
		{LibraryID: libraryID, MediaItemID: &second.ID, StoragePath: second.SourcePath, StableIdentityKey: "stable:shared-b", ProviderName: "local", ProbeStatus: "complete", DurationSeconds: &durationSeconds},
	} {
		legacyFile := file
		if err := svc.db.WithContext(ctx).Create(&legacyFile).Error; err != nil {
			t.Fatalf("create legacy file %q: %v", legacyFile.StoragePath, err)
		}
	}

	if err := svc.backfillSeries(ctx, run); err != nil {
		t.Fatalf("backfill series: %v", err)
	}

	finalized, err := svc.finalizeLegacyBackfillRun(ctx, run.ID, LegacyBackfillStatusCompleted, "")
	if err != nil {
		t.Fatalf("finalize run: %v", err)
	}
	if finalized.SuccessCount != 2 || finalized.ConflictCount != 1 {
		t.Fatalf("unexpected finalized counts: %#v", finalized)
	}

	var seriesItems []database.CatalogItem
	if err := svc.db.WithContext(ctx).Where("library_id = ? AND type = ?", libraryID, ItemTypeSeries).Order("id asc").Find(&seriesItems).Error; err != nil {
		t.Fatalf("list series items: %v", err)
	}
	if len(seriesItems) != 2 {
		t.Fatalf("expected separate provider-backed series rows, got %#v", seriesItems)
	}

	var externalIDs []database.CatalogExternalID
	if err := svc.db.WithContext(ctx).Where("provider_type = ?", "series").Order("external_id asc").Find(&externalIDs).Error; err != nil {
		t.Fatalf("list series external ids: %v", err)
	}
	if len(externalIDs) != 2 || externalIDs[0].ExternalID != "tv:100" || externalIDs[1].ExternalID != "tv:200" {
		t.Fatalf("expected both provider identities to persist, got %#v", externalIDs)
	}
}

func TestLegacyBackfillSeriesPreservesProviderEvidenceFromProviderRow(t *testing.T) {
	svc, ctx := newTestService(t)

	libraryID := uint(14)
	run, err := svc.createLegacyBackfillRun(ctx, LegacyBackfillScope{Kind: LegacyBackfillScopeLibrary, LibraryID: &libraryID}, 80)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	confidence := 0.88
	year := 2022
	seasonNumber := 1
	firstEpisode := 1
	secondEpisode := 2
	runtimeSeconds := 2400
	providerRow := database.MediaItem{
		LibraryID:          libraryID,
		Type:               ItemTypeEpisode,
		Title:              "Episode One",
		SeriesTitle:        "Provider Evidence Show",
		Year:               &year,
		SeasonNumber:       &seasonNumber,
		EpisodeNumber:      &firstEpisode,
		SourcePath:         "/library/shows/provider-evidence/Season 01/episode-one.mkv",
		MatchStatus:        "matched",
		MetadataProvider:   "tmdb",
		ExternalID:         "tv:555",
		MetadataConfidence: &confidence,
		RuntimeSeconds:     &runtimeSeconds,
		Status:             "ready",
	}
	fallbackRow := database.MediaItem{
		LibraryID:      libraryID,
		Type:           ItemTypeEpisode,
		Title:          "Episode Two",
		SeriesTitle:    "Provider Evidence Show",
		Overview:       "Richer overview",
		PosterURL:      "https://images.example.com/provider-evidence/poster.jpg",
		BackdropURL:    "https://images.example.com/provider-evidence/backdrop.jpg",
		LogoURL:        "https://images.example.com/provider-evidence/logo.png",
		Year:           &year,
		SeasonNumber:   &seasonNumber,
		EpisodeNumber:  &secondEpisode,
		SourcePath:     "/library/shows/provider-evidence/Season 01/episode-two.mkv",
		MatchStatus:    "matched",
		RuntimeSeconds: &runtimeSeconds,
		Status:         "ready",
	}
	for _, item := range []*database.MediaItem{&providerRow, &fallbackRow} {
		if err := svc.db.WithContext(ctx).Create(item).Error; err != nil {
			t.Fatalf("create legacy episode %q: %v", item.SourcePath, err)
		}
	}

	durationSeconds := 2400.0
	for _, file := range []database.MediaFile{
		{LibraryID: libraryID, MediaItemID: &providerRow.ID, StoragePath: providerRow.SourcePath, StableIdentityKey: "stable:provider-evidence:1", ProviderName: "local", ProbeStatus: "complete", DurationSeconds: &durationSeconds},
		{LibraryID: libraryID, MediaItemID: &fallbackRow.ID, StoragePath: fallbackRow.SourcePath, StableIdentityKey: "stable:provider-evidence:2", ProviderName: "local", ProbeStatus: "complete", DurationSeconds: &durationSeconds},
	} {
		legacyFile := file
		if err := svc.db.WithContext(ctx).Create(&legacyFile).Error; err != nil {
			t.Fatalf("create legacy file %q: %v", legacyFile.StoragePath, err)
		}
	}

	if err := svc.backfillSeries(ctx, run); err != nil {
		t.Fatalf("backfill series: %v", err)
	}

	var externalIDs []database.CatalogExternalID
	if err := svc.db.WithContext(ctx).Where("provider = ? AND provider_type = ?", "tmdb", "series").Find(&externalIDs).Error; err != nil {
		t.Fatalf("list series external ids: %v", err)
	}
	if len(externalIDs) != 1 || externalIDs[0].ExternalID != providerRow.ExternalID {
		t.Fatalf("expected provider identity from provider row, got %#v", externalIDs)
	}

	var sources []database.MetadataSource
	if err := svc.db.WithContext(ctx).Where("source_name = ?", "tmdb").Find(&sources).Error; err != nil {
		t.Fatalf("list provider metadata sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected provider metadata source, got %#v", sources)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(sources[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("decode provider metadata payload: %v", err)
	}
	if payload["legacy_media_item_id"] != float64(providerRow.ID) {
		t.Fatalf("expected provider payload to reference provider row %d, got %#v", providerRow.ID, payload)
	}
}
