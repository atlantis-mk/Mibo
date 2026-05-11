package library

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/recognition"
)

func TestDirectoryReductionContextEvidenceGroupsSiblingMovieVersions(t *testing.T) {
	year := 2024
	files := []database.InventoryFile{
		{ID: 1, StoragePath: "/library/Movie.2024.1080p.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 2, StoragePath: "/library/Movie.2024.2160p.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 3, StoragePath: "/library/poster.jpg", ContentClass: SourceContentClassOther},
	}
	signals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &files[0].ID, TitleCandidate: "Movie", Year: &year, Quality: "1080p"},
		2: {InventoryFileID: &files[1].ID, TitleCandidate: "Movie", Year: &year, Quality: "2160p"},
	}
	evidence := directoryReductionContextEvidence(files, signals)
	if len(evidence[1]) == 0 || len(evidence[2]) == 0 {
		t.Fatalf("expected reduction evidence for both movie versions, got %#v", evidence)
	}
	if evidence[1][0].Assignment != "movie_multi_version" || evidence[2][0].ParentKey != recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "Movie", Year: &year}) {
		t.Fatalf("unexpected movie reduction evidence %#v", evidence)
	}
	if evidence[1][0].VariantKey == evidence[2][0].VariantKey {
		t.Fatalf("expected distinct variant keys, got %#v", evidence)
	}
	residual, _ := evidence[1][0].Payload["residual_paths"].([]string)
	if len(residual) != 1 || residual[0] != "/library/poster.jpg" {
		t.Fatalf("expected poster residual, got %#v", evidence[1][0].Payload)
	}
}

func TestDirectoryReductionContextEvidenceGroupsSiblingEpisodeVersions(t *testing.T) {
	season := 1
	episode := 2
	files := []database.InventoryFile{
		{ID: 1, StoragePath: "/library/Show.S01E02.1080p.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 2, StoragePath: "/library/Show.S01E02.2160p.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 3, StoragePath: "/library/Show.S01E03.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
	}
	signals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &files[0].ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: &episode, Quality: "1080p"},
		2: {InventoryFileID: &files[1].ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: &episode, Quality: "2160p"},
		3: {InventoryFileID: &files[2].ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: intPtrForReduction(3)},
	}
	evidence := directoryReductionContextEvidence(files, signals)
	if len(evidence[1]) == 0 || evidence[1][0].Assignment != "episode_multi_version" {
		t.Fatalf("expected episode multi-version evidence, got %#v", evidence)
	}
	residual, _ := evidence[1][0].Payload["residual_paths"].([]string)
	if len(residual) != 1 || residual[0] != "/library/Show.S01E03.mkv" {
		t.Fatalf("expected leftover episode in residuals, got %#v", evidence[1][0].Payload)
	}
}

func TestDirectoryReductionDecisionForMovieCollectionAndSeries(t *testing.T) {
	yearA := 2024
	yearB := 2025
	season := 1
	movieFiles := []database.InventoryFile{
		{ID: 1, StoragePath: "/library/A.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 2, StoragePath: "/library/B.2025.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
	}
	movieSignals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &movieFiles[0].ID, TitleCandidate: "A", Year: &yearA},
		2: {InventoryFileID: &movieFiles[1].ID, TitleCandidate: "B", Year: &yearB},
	}
	movieDecision, ok := directoryReductionDecisionForFiles(movieFiles, movieSignals)
	if !ok || movieDecision.Interpretation != pathTreeWorkGroupShapeMovieCollection {
		t.Fatalf("expected movie collection decision, got %#v ok=%v", movieDecision, ok)
	}
	seriesFiles := []database.InventoryFile{
		{ID: 3, StoragePath: "/library/Show.S01E01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 4, StoragePath: "/library/Show.S01E02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
	}
	seriesSignals := map[uint]database.InventoryFileSignal{
		3: {InventoryFileID: &seriesFiles[0].ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: intPtrForReduction(1)},
		4: {InventoryFileID: &seriesFiles[1].ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: intPtrForReduction(2)},
	}
	seriesDecision, ok := directoryReductionDecisionForFiles(seriesFiles, seriesSignals)
	if !ok || seriesDecision.Interpretation != pathTreeWorkGroupShapeSeries {
		t.Fatalf("expected series decision, got %#v ok=%v", seriesDecision, ok)
	}
	movieContext := residualDirectoryContextEvidence(movieFiles, movieSignals, movieDecision)
	if len(movieContext[1]) == 0 || len(movieContext[2]) == 0 {
		t.Fatalf("expected movie collection context evidence, got %#v", movieContext)
	}
	if movieContext[1][0].Assignment != pathTreeWorkGroupShapeMovieCollection || movieContext[1][0].ParentKey == movieContext[2][0].ParentKey {
		t.Fatalf("expected distinct movie parent keys from collection context, got %#v", movieContext)
	}
	seriesContext := residualDirectoryContextEvidence(seriesFiles, seriesSignals, seriesDecision)
	if len(seriesContext[3]) == 0 || seriesContext[3][0].Assignment != pathTreeWorkGroupShapeSeries {
		t.Fatalf("expected series context evidence, got %#v", seriesContext)
	}
}

func TestDirectoryReductionContextEvidenceTreatsLeadingNumericAsMovieCollection(t *testing.T) {
	one := 1
	two := 2
	files := []database.InventoryFile{
		{ID: 1, StoragePath: "/library/1-bdg01-suzanna-egals-11_hq.mp4", Container: "mp4", ContentClass: SourceContentClassVideo},
		{ID: 2, StoragePath: "/library/2-silvia-saint_hq.mp4", Container: "mp4", ContentClass: SourceContentClassVideo},
	}
	signals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &files[0].ID, TitleCandidate: "1 bdg01 suzanna egals 11 hq", EpisodeNumber: &one, EpisodeSource: "leading_numeric"},
		2: {InventoryFileID: &files[1].ID, TitleCandidate: "2 silvia saint hq", EpisodeNumber: &two, EpisodeSource: "leading_numeric"},
	}
	evidence := directoryReductionContextEvidence(files, signals)
	if len(evidence[1]) == 0 || len(evidence[2]) == 0 {
		t.Fatalf("expected movie collection evidence for leading numeric files, got %#v", evidence)
	}
	if evidence[1][0].Assignment != pathTreeWorkGroupShapeMovieCollection || evidence[2][0].Assignment != pathTreeWorkGroupShapeMovieCollection {
		t.Fatalf("expected movie collection assignments, got %#v", evidence)
	}
}

func TestPersistRecognitionManifestForFilesSavesDirectoryReductionDecision(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	svc := NewService(config.Config{}, db, nil, nil)
	libraryRecord := database.Library{ID: 1, MediaSourceID: 1, RootPath: "/library"}
	if err := db.WithContext(ctx).Create(&libraryRecord).Error; err != nil {
		t.Fatalf("create library: %v", err)
	}
	year := 2024
	files := []database.InventoryFile{
		{ID: 1, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie.2024.1080p.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, LibraryID: libraryRecord.ID, MediaSourceID: libraryRecord.MediaSourceID, StorageProvider: "local", StoragePath: "/library/Movie.2024.2160p.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	for _, file := range files {
		if err := db.WithContext(ctx).Create(&file).Error; err != nil {
			t.Fatalf("create file: %v", err)
		}
		model := extractFilenameSignalModel(file.StoragePath)
		model.Identity.TitleCandidate = "Movie"
		model.Identity.Year = &year
		model.ReleaseHints.Quality = map[uint]string{1: "1080p", 2: "2160p"}[file.ID]
		if err := saveInventoryFileSignals(ctx, db, inventoryFileSignalScope{LibraryID: libraryRecord.ID, StorageProvider: "local", ClassifierVersion: contentShapeSettingsFromConfig(config.Config{}).ClassifierVersion}, []inventoryFileSignalInput{{File: file, Model: model}}); err != nil {
			t.Fatalf("create signal: %v", err)
		}
	}
	if _, err := svc.persistRecognitionManifestForFiles(ctx, libraryRecord, files, "/library"); err != nil {
		t.Fatalf("persist manifest: %v", err)
	}
	var decision database.ClassificationDecision
	if err := db.WithContext(ctx).Where("library_id = ? AND decision_type = ?", libraryRecord.ID, "directory_reduction").First(&decision).Error; err != nil {
		t.Fatalf("load directory reduction decision: %v", err)
	}
	if decision.CandidateType != "movie_multi_version" || decision.Status != "provisional" {
		t.Fatalf("unexpected directory reduction decision %#v", decision)
	}
}

func TestDirectoryReductionDecisionClassifiesReviewSubtypes(t *testing.T) {
	year := 2024
	filesWithAttachments := []database.InventoryFile{
		{ID: 1, StoragePath: "/library/Movie.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 2, StoragePath: "/library/poster.jpg", ContentClass: SourceContentClassOther},
	}
	signalsWithAttachments := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &filesWithAttachments[0].ID, TitleCandidate: "Movie", Year: &year},
	}
	decision, ok := directoryReductionDecisionForFiles(filesWithAttachments, signalsWithAttachments)
	if !ok || decision.Interpretation != pathTreeWorkGroupShapeReview || decision.Evidence["review_subtype"] != directoryReductionReviewSingleWorkWithNoise {
		t.Fatalf("expected single work with noise review subtype, got %#v ok=%v", decision, ok)
	}
	context := singleIdentityResidualContextEvidence(filesWithAttachments, signalsWithAttachments, decision)
	if len(context[1]) == 0 || context[1][0].Assignment != directoryReductionAssignmentSingleWorkIdentity {
		t.Fatalf("expected single work identity context evidence, got %#v", context)
	}
	filesWithExtras := []database.InventoryFile{
		{ID: 3, StoragePath: "/library/Movie.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
		{ID: 4, StoragePath: "/library/Movie.2024.trailer.mkv", Container: "mkv", ContentClass: SourceContentClassVideo},
	}
	signalsWithExtras := map[uint]database.InventoryFileSignal{
		3: {InventoryFileID: &filesWithExtras[0].ID, TitleCandidate: "Movie", Year: &year},
		4: {InventoryFileID: &filesWithExtras[1].ID, TitleCandidate: "Movie", Year: &year, Role: "trailer", IsExtra: true},
	}
	decision, ok = directoryReductionDecisionForFiles(filesWithExtras, signalsWithExtras)
	if !ok || decision.Evidence["review_subtype"] != directoryReductionReviewExtrasMixed {
		t.Fatalf("expected extras mixed review subtype, got %#v ok=%v", decision, ok)
	}
	excluded := directoryReductionExcludedFileIDs(filesWithExtras, signalsWithExtras)
	if excluded[4] != "directory_reduction_extras" || excluded[3] != "" {
		t.Fatalf("expected only extras file excluded, got %#v", excluded)
	}
	attachmentExcluded := directoryReductionExcludedFileIDs(filesWithAttachments, signalsWithAttachments)
	if attachmentExcluded != nil {
		t.Fatalf("expected single-work-with-noise attachments to remain for now, got %#v", attachmentExcluded)
	}
}

func TestDirectoryReductionDecisionMarksArtworkSafeAttachments(t *testing.T) {
	files := []database.InventoryFile{{ID: 1, StoragePath: "/library/poster.jpg", ContentClass: SourceContentClassOther}, {ID: 2, StoragePath: "/library/fanart.png", ContentClass: SourceContentClassOther}}
	decision, ok := directoryReductionDecisionForFiles(files, map[uint]database.InventoryFileSignal{})
	if !ok || decision.Evidence["review_subtype"] != directoryReductionReviewAttachmentsOnly || decision.Evidence["artwork_safe"] != true {
		t.Fatalf("expected artwork-safe attachments decision, got %#v ok=%v", decision, ok)
	}
}

func intPtrForReduction(value int) *int {
	return &value
}
