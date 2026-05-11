package recognition

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
)

func TestMaterializeMetadataCreatesMovieOnce(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	materializer := NewMaterializer(db)
	candidate := database.RecognitionCandidate{ID: 1, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie:2024"}
	decision := database.RecognitionDecision{CandidateID: &candidate.ID, TargetKind: candidate.CandidateType, TargetKey: candidate.CandidateKey, Outcome: DecisionOutcomeAccepted}
	graph := ManifestGraph{Candidates: []database.RecognitionCandidate{candidate}}
	first, err := materializer.MaterializeMetadata(ctx, graph, []database.RecognitionDecision{decision})
	if err != nil {
		t.Fatalf("materialize first: %v", err)
	}
	second, err := materializer.MaterializeMetadata(ctx, graph, []database.RecognitionDecision{decision})
	if err != nil {
		t.Fatalf("materialize second: %v", err)
	}
	if len(first.MetadataIDs) != 1 || len(second.MetadataIDs) != 1 || first.MetadataIDs[0] != second.MetadataIDs[0] {
		t.Fatalf("expected idempotent metadata ids, first=%#v second=%#v", first.MetadataIDs, second.MetadataIDs)
	}
	if len(first.ProjectionMetadataIDs) != 1 || first.ProjectionMetadataIDs[0] != first.MetadataIDs[0] {
		t.Fatalf("expected projection metadata ids, got %#v", first.ProjectionMetadataIDs)
	}
	var count int64
	if err := db.Model(&database.MetadataItem{}).Count(&count).Error; err != nil {
		t.Fatalf("count metadata: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one metadata item, got %d", count)
	}
}

func TestMaterializeResourcesCreatesResourceAndLinks(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.mkv", ContentClass: "video", Status: "available"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	materializer := NewMaterializer(db)
	work := database.RecognitionCandidate{ID: 1, CandidateKey: "work:movie:movie:2024", CandidateType: CandidateTypeWork, CandidateRole: WorkKindMovie, CanonicalKey: "work:movie:movie:2024"}
	resource := database.RecognitionCandidate{ID: 2, CandidateKey: "playable_resource:local:path:/library/Movie.mkv", CandidateType: CandidateTypePlayableResource, ParentCandidateKey: work.CandidateKey, PrimaryInventoryID: &file.ID, ResourceShape: ResourceKindSingleFile, VariantKey: "variant:2160p", EditionKey: "edition:directors-cut", EvidenceJSON: `{"source":"test"}`}
	graph := ManifestGraph{Manifest: database.RecognitionManifest{ID: 1, LibraryID: 1}, Candidates: []database.RecognitionCandidate{work, resource}}
	metadataDecision := database.RecognitionDecision{CandidateID: &work.ID, TargetKind: work.CandidateType, TargetKey: work.CandidateKey, Outcome: DecisionOutcomeAccepted}
	if _, err := materializer.MaterializeMetadata(ctx, graph, []database.RecognitionDecision{metadataDecision}); err != nil {
		t.Fatalf("materialize metadata: %v", err)
	}
	resourceDecision := database.RecognitionDecision{CandidateID: &resource.ID, TargetKind: resource.CandidateType, TargetKey: resource.CandidateKey, Outcome: DecisionOutcomeAccepted}
	result, err := materializer.MaterializeResources(ctx, graph, []database.RecognitionDecision{resourceDecision})
	if err != nil {
		t.Fatalf("materialize resources: %v", err)
	}
	if len(result.ResourceIDs) != 1 {
		t.Fatalf("expected one resource id, got %#v", result.ResourceIDs)
	}
	if len(result.ProjectionResourceIDs) != 1 || result.ProjectionResourceIDs[0] != result.ResourceIDs[0] {
		t.Fatalf("expected projection resource ids, got %#v", result.ProjectionResourceIDs)
	}
	var linkCount int64
	if err := db.Model(&database.ResourceMetadataLink{}).Count(&linkCount).Error; err != nil {
		t.Fatalf("count metadata links: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("expected resource metadata link, got %d", linkCount)
	}
	var link database.ResourceMetadataLink
	if err := db.First(&link).Error; err != nil {
		t.Fatalf("load metadata link: %v", err)
	}
	if link.Role != database.ResourceLinkRoleVersion || link.EvidenceJSON == "" {
		t.Fatalf("expected version link with evidence, got %#v", link)
	}
	second, err := materializer.MaterializeResources(ctx, graph, []database.RecognitionDecision{resourceDecision})
	if err != nil {
		t.Fatalf("materialize resources second: %v", err)
	}
	if len(second.ResourceIDs) != 1 || second.ResourceIDs[0] != result.ResourceIDs[0] {
		t.Fatalf("expected idempotent resource ids, first=%#v second=%#v", result.ResourceIDs, second.ResourceIDs)
	}
	var resourceCount int64
	if err := db.Model(&database.Resource{}).Count(&resourceCount).Error; err != nil {
		t.Fatalf("count resources: %v", err)
	}
	if resourceCount != 1 {
		t.Fatalf("expected one resource after rerun, got %d", resourceCount)
	}
}

func TestMovieCollectionManifestMaterializesResourceLinksAfterRepositorySave(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	episode := 1
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/1-cwdv-027-shiori-uehara-catwalk-poison-27_hq.mp4", Container: "mp4", ContentClass: "video", Status: "available"},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/1-cwdv-028-ryoko-murakami-catwalk-poison-28_hq.mp4", Container: "mp4", ContentClass: "video", Status: "available"},
	}
	if err := db.Create(&files).Error; err != nil {
		t.Fatalf("create files: %v", err)
	}
	signals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &files[0].ID, TitleCandidate: "1 cwdv 027 shiori uehara catwalk poison 27 hq", EpisodeNumber: &episode, EpisodeSource: "leading_numeric"},
		2: {InventoryFileID: &files[1].ID, TitleCandidate: "1 cwdv 028 ryoko murakami catwalk poison 28 hq", EpisodeNumber: &episode, EpisodeSource: "leading_numeric"},
	}
	firstKey := MovieWorkKey(MovieWorkInput{Title: signals[1].TitleCandidate})
	secondKey := MovieWorkKey(MovieWorkInput{Title: signals[2].TitleCandidate})
	build := BuildManifestFromInventory(ManifestBuildInput{
		Scope:       ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"},
		Files:       files,
		FileSignals: signals,
		ContextEvidence: map[uint][]ContextEvidence{
			1: {{Source: "directory_reduction", Assignment: "movie_collection", ParentKey: firstKey, TargetKey: "/library", ReviewState: "auto"}},
			2: {{Source: "directory_reduction", Assignment: "movie_collection", ParentKey: secondKey, TargetKey: "/library", ReviewState: "auto"}},
		},
	})
	repo := NewRepository(db)
	manifest, err := repo.UpsertManifest(ctx, build.ManifestScope)
	if err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}
	for idx := range build.Candidates {
		build.Candidates[idx].ManifestID = manifest.ID
	}
	for idx := range build.Evidence {
		build.Evidence[idx].ManifestID = manifest.ID
	}
	if err := repo.SaveCandidates(ctx, build.Candidates); err != nil {
		t.Fatalf("save candidates: %v", err)
	}
	if err := repo.SaveEvidence(ctx, build.Evidence); err != nil {
		t.Fatalf("save evidence: %v", err)
	}
	graph, err := repo.LoadManifestGraph(ctx, manifest.ID)
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	resolved := NewResolver(nil).Resolve(graph)
	materializer := NewMaterializer(db)
	if _, err := materializer.MaterializeMetadata(ctx, graph, resolved.Decisions); err != nil {
		t.Fatalf("materialize metadata: %v", err)
	}
	if _, err := materializer.MaterializeResources(ctx, graph, resolved.Decisions); err != nil {
		t.Fatalf("materialize resources: %v", err)
	}

	var linkCount int64
	if err := db.Model(&database.ResourceMetadataLink{}).Count(&linkCount).Error; err != nil {
		t.Fatalf("count metadata links: %v", err)
	}
	if linkCount != 2 {
		t.Fatalf("expected resource metadata links for both movie collection files, got %d", linkCount)
	}
}

func TestMaterializeMetadataCreatesSeriesSeasonEpisodeHierarchy(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "mibo.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	materializer := NewMaterializer(db)
	series := database.RecognitionCandidate{ID: 1, CandidateKey: "work:series:show", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeries, CanonicalKey: "work:series:show", EvidenceJSON: `{"title":"Show"}`}
	season := database.RecognitionCandidate{ID: 2, CandidateKey: "work:season:work:series:show:s01", CandidateType: CandidateTypeWork, CandidateRole: WorkKindSeason, ParentCandidateKey: series.CandidateKey, CanonicalKey: "work:season:work:series:show:s01", EvidenceJSON: `{"title":"Show","season_number":1}`}
	episode := database.RecognitionCandidate{ID: 3, CandidateKey: "episode:work:series:show:s01:e01", CandidateType: CandidateTypeEpisode, CandidateRole: WorkKindEpisode, ParentCandidateKey: season.CandidateKey, CanonicalKey: "episode:work:series:show:s01:e01", EvidenceJSON: `{"title":"Show","season_number":1,"episode_number":1}`}
	graph := ManifestGraph{Candidates: []database.RecognitionCandidate{series, season, episode}}
	decisions := []database.RecognitionDecision{
		{CandidateID: &series.ID, TargetKind: series.CandidateType, TargetKey: series.CandidateKey, Outcome: DecisionOutcomeAccepted},
		{CandidateID: &season.ID, TargetKind: season.CandidateType, TargetKey: season.CandidateKey, Outcome: DecisionOutcomeAccepted},
		{CandidateID: &episode.ID, TargetKind: episode.CandidateType, TargetKey: episode.CandidateKey, Outcome: DecisionOutcomeAccepted},
	}
	if _, err := materializer.MaterializeMetadata(ctx, graph, decisions); err != nil {
		t.Fatalf("materialize hierarchy: %v", err)
	}
	var rows []database.MetadataItem
	if err := db.WithContext(ctx).Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("load metadata items: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected series, season, episode rows, got %#v", rows)
	}
	var gotSeries, gotSeason, gotEpisode database.MetadataItem
	for _, row := range rows {
		switch row.ItemType {
		case database.MetadataItemTypeSeries:
			gotSeries = row
		case database.MetadataItemTypeSeason:
			gotSeason = row
		case database.MetadataItemTypeEpisode:
			gotEpisode = row
		}
	}
	if gotSeries.ID == 0 || gotSeason.ID == 0 || gotEpisode.ID == 0 {
		t.Fatalf("expected hierarchy rows, got %#v", rows)
	}
	if gotSeason.ParentID == nil || *gotSeason.ParentID != gotSeries.ID || gotSeason.RootID == nil || *gotSeason.RootID != gotSeries.ID {
		t.Fatalf("expected season under series, got season=%#v series=%#v", gotSeason, gotSeries)
	}
	if gotEpisode.ParentID == nil || *gotEpisode.ParentID != gotSeason.ID || gotEpisode.RootID == nil || *gotEpisode.RootID != gotSeries.ID {
		t.Fatalf("expected episode under season with series root, got episode=%#v season=%#v series=%#v", gotEpisode, gotSeason, gotSeries)
	}
	if gotSeason.IndexNumber == nil || *gotSeason.IndexNumber != 1 {
		t.Fatalf("expected season index 1, got %#v", gotSeason)
	}
	if gotEpisode.ParentIndexNumber == nil || *gotEpisode.ParentIndexNumber != 1 || gotEpisode.IndexNumber == nil || *gotEpisode.IndexNumber != 1 {
		t.Fatalf("expected episode numbering S01E01, got %#v", gotEpisode)
	}
}
