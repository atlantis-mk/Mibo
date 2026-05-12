package recognition

import (
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestBuildManifestFromInventoryCreatesPlayableResourceCandidates(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.mkv", StableIdentityKey: "stable-movie", ContentClass: "video", Status: "available", HashesJSON: `{"md5":"same"}`}
	year := 2024
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Movie", Year: &year}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}})
	if output.ManifestScope.ManifestKey == "" || output.ManifestScope.Fingerprint == "" {
		t.Fatalf("expected manifest scope keys, got %#v", output.ManifestScope)
	}
	candidate, ok := candidateByType(output.Candidates, CandidateTypePlayableResource)
	if !ok {
		t.Fatalf("expected playable candidate, got %#v", output.Candidates)
	}
	if candidate.CandidateType != CandidateTypePlayableResource || candidate.CandidateKey != "playable_resource:local:stable:stable-movie" {
		t.Fatalf("unexpected candidate %#v", candidate)
	}
	if len(output.Evidence) < 3 {
		t.Fatalf("expected inventory evidence, got %#v", output.Evidence)
	}
}

func TestConstructGraphFromInventoryReturnsManifestCandidatesAndEvidence(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.mkv", StableIdentityKey: "stable-movie", ContentClass: "video", Status: "available"}
	year := 2024
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Movie", Year: &year}
	output := ConstructGraphFromInventory(GraphConstructInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}})
	if output.ManifestScope.ManifestKey == "" {
		t.Fatalf("expected graph constructor manifest scope, got %#v", output.ManifestScope)
	}
	if len(output.Candidates) == 0 || len(output.Evidence) == 0 {
		t.Fatalf("expected graph constructor output candidates/evidence, got %#v", output)
	}
}

func TestConstructGraphFromInventoryKeepsWeakMovieSignalInReview(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.mkv", StableIdentityKey: "weak-movie", ContentClass: "video", Status: "available"}
	output := ConstructGraphFromInventory(GraphConstructInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}})
	for _, candidate := range output.Candidates {
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie {
			t.Fatalf("did not expect weak movie signal to materialize work candidate, got %#v", output.Candidates)
		}
	}
	seenReview := false
	for _, classification := range output.MediaGraphClassifications {
		if classification.GroupKind == mediaGroupKindMoviePackage && classification.ReviewState == database.ReviewStateNeedsReview {
			seenReview = true
		}
	}
	if !seenReview {
		t.Fatalf("expected weak movie package review classification, got %#v", output.MediaGraphClassifications)
	}
	if len(output.Evidence) == 0 {
		t.Fatalf("expected weak signal evidence to be retained")
	}
}

func TestConstructGraphFromInventoryAcceptsEpisodeRunGroup(t *testing.T) {
	season := 1
	episodeOne := 1
	episodeTwo := 2
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 1/01.mkv", StableIdentityKey: "ep-1", ContentClass: "video", Status: "available"},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 1/02.mkv", StableIdentityKey: "ep-2", ContentClass: "video", Status: "available"},
	}
	signals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &files[0].ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: &episodeOne},
		2: {InventoryFileID: &files[1].ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: &episodeTwo},
	}
	output := ConstructGraphFromInventory(GraphConstructInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library/Show/Season 1", ClassifierVersion: "test"}, Files: files, FileSignals: signals})
	seenAcceptedRun := false
	for _, classification := range output.MediaGraphClassifications {
		if classification.GroupKind == mediaGroupKindEpisodeRun && classification.ReviewState == database.ReviewStateAccepted {
			seenAcceptedRun = true
		}
	}
	if !seenAcceptedRun {
		t.Fatalf("expected accepted episode run classification, got %#v", output.MediaGraphClassifications)
	}
	resource, ok := candidateByType(output.Candidates, CandidateTypePlayableResource)
	if !ok || resource.ParentCandidateKey != EpisodeKey(EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 1}) {
		t.Fatalf("expected resource linked to accepted episode, got %#v", output.Candidates)
	}
	for _, candidate := range output.Candidates {
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie {
			t.Fatalf("did not expect episode run to materialize movie candidate, got %#v", output.Candidates)
		}
	}
}

func TestBuildManifestFromInventoryMarksExcludedFile(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/sample.mkv", ContentClass: "video", Status: "available"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, ExcludedFileIDs: map[uint]string{1: "sample_excluded"}})
	candidate, ok := candidateByType(output.Candidates, CandidateTypePlayableResource)
	if !ok || candidate.ReviewState != database.ReviewStateRejected {
		t.Fatalf("expected rejected excluded candidate, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryMarksDirectoryReductionExcludedFile(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.trailer.mkv", ContentClass: "video", Status: "available"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, ExcludedFileIDs: map[uint]string{1: "directory_reduction_extras"}})
	candidate, ok := candidateByType(output.Candidates, CandidateTypePlayableResource)
	if !ok || candidate.ReviewState != database.ReviewStateRejected || candidate.CandidateRole != "excluded" {
		t.Fatalf("expected directory reduction exclusion to reject candidate, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryAddsFilenameSignalEvidence(t *testing.T) {
	year := 2024
	episode := 2
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show.S01E02.2160p.mkv", ContentClass: "video", Status: "available"}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Show", Year: &year, EpisodeNumber: &episode, Quality: "2160p", Codec: "x265", EvidenceJSON: `[{"kind":"quality","value":"2160p"}]`}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}})
	seenQuality := false
	seenEpisode := false
	for _, evidence := range output.Evidence {
		if evidence.EvidenceKey == "quality" && evidence.EvidenceValue == "2160p" {
			seenQuality = true
		}
		if evidence.EvidenceKey == "episode_number" && evidence.EvidenceValue == "2" {
			seenEpisode = true
		}
	}
	if !seenQuality || !seenEpisode {
		t.Fatalf("expected signal evidence quality=%v episode=%v in %#v", seenQuality, seenEpisode, output.Evidence)
	}
}

func candidateByType(candidates []database.RecognitionCandidate, candidateType string) (database.RecognitionCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.CandidateType == candidateType {
			return candidate, true
		}
	}
	return database.RecognitionCandidate{}, false
}

func TestBuildManifestFromInventoryAddsSidecarHintEvidence(t *testing.T) {
	year := 2024
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.mkv", ContentClass: "video", Status: "available"}
	hint := SidecarHint{Path: "/library/Movie.2024.nfo", Extension: ".nfo", ParseStatus: "parsed", MediaType: "movie", Title: "Movie", Year: &year, ExternalIDs: map[string]string{"tmdb": "123"}}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, SidecarHints: map[uint][]SidecarHint{1: {hint}}})
	seenExternal := false
	seenTitle := false
	for _, evidence := range output.Evidence {
		if evidence.EvidenceKey == "external_id:tmdb" && evidence.EvidenceValue == "123" {
			seenExternal = true
		}
		if evidence.EvidenceKey == "title" && evidence.EvidenceValue == "Movie" {
			seenTitle = true
		}
	}
	if !seenExternal || !seenTitle {
		t.Fatalf("expected sidecar title/external evidence, got %#v", output.Evidence)
	}
}

func TestBuildManifestFromInventoryAddsDirectoryContextEvidence(t *testing.T) {
	confidence := 0.88
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2160p.mkv", ContentClass: "video", Status: "available"}
	contextEvidence := ContextEvidence{Source: evidenceSourcePathTree, Assignment: "movie_version", TargetKey: "/library/Movie (2024)", ReviewState: "auto", Confidence: &confidence}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, ContextEvidence: map[uint][]ContextEvidence{1: {contextEvidence}}})
	seenAssignment := false
	for _, evidence := range output.Evidence {
		if evidence.EvidenceKind == evidenceKindDirectoryContext && evidence.EvidenceSource == evidenceSourcePathTree && evidence.EvidenceKey == "assignment" && evidence.EvidenceValue == "movie_version" {
			seenAssignment = true
		}
	}
	if !seenAssignment {
		t.Fatalf("expected directory context assignment evidence, got %#v", output.Evidence)
	}
}

func TestBuildManifestFromInventoryAddsGroupedCandidates(t *testing.T) {
	year := 2024
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.2160p.mkv", Container: "mkv", ContentClass: "video", Status: "available", HashesJSON: `{"md5":"same-binary"}`}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Movie", Year: &year, Quality: "2160p", Edition: "Directors Cut", SourceTagsJSON: `["BluRay"]`}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}})
	seenWork := false
	seenVariant := false
	seenEdition := false
	seenDuplicate := false
	for _, candidate := range output.Candidates {
		switch candidate.CandidateType {
		case CandidateTypeWork:
			seenWork = true
		case CandidateTypeVariant:
			seenVariant = true
		case CandidateTypeEdition:
			seenEdition = true
		case CandidateTypeDuplicateBinary:
			seenDuplicate = true
		}
	}
	if !seenWork || !seenVariant || !seenEdition || !seenDuplicate {
		t.Fatalf("expected grouped candidates work=%v variant=%v edition=%v duplicate=%v in %#v", seenWork, seenVariant, seenEdition, seenDuplicate, output.Candidates)
	}
}

func TestBuildManifestFromInventoryUsesOnlyProvidedLocalFacts(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.mkv", ContentClass: "video", Status: "available"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}})
	for _, evidence := range output.Evidence {
		switch evidence.EvidenceSource {
		case evidenceSourceInventory, evidenceSourceResource, evidenceSourceSignal, evidenceSourceSidecar, evidenceSourceContentShape, evidenceSourcePathTree, evidenceSourceExclusion:
		default:
			t.Fatalf("unexpected non-local evidence source %q in %#v", evidence.EvidenceSource, evidence)
		}
	}
}

func TestBuildManifestFromInventoryGroupingIsOrderIndependent(t *testing.T) {
	year := 2024
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.1080p.mkv", Container: "mkv", ContentClass: "video", Status: "available"},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.2024.2160p.mkv", Container: "mkv", ContentClass: "video", Status: "available"},
	}
	signals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &files[0].ID, TitleCandidate: "Movie", Year: &year, Quality: "1080p"},
		2: {InventoryFileID: &files[1].ID, TitleCandidate: "Movie", Year: &year, Quality: "2160p"},
	}
	forward := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: files, FileSignals: signals})
	reverse := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{files[1], files[0]}, FileSignals: signals})
	if candidateTypeCounts(forward.Candidates)[CandidateTypeWork] != candidateTypeCounts(reverse.Candidates)[CandidateTypeWork] || candidateTypeCounts(forward.Candidates)[CandidateTypeVariant] != candidateTypeCounts(reverse.Candidates)[CandidateTypeVariant] {
		t.Fatalf("expected order-independent grouping, forward=%#v reverse=%#v", forward.Candidates, reverse.Candidates)
	}
}

func TestBuildManifestFromInventoryUsesDirectoryReductionParentAndVariantForEpisode(t *testing.T) {
	season := 1
	episode := 2
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show.S01E02.2160p.mkv", Container: "mkv", ContentClass: "video", Status: "available"}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: &episode}
	contextParentKey := EpisodeKey(EpisodeInput{SeriesTitle: "Show", SeasonNumber: season, EpisodeNumber: episode})
	context := ContextEvidence{Source: "directory_reduction", Assignment: "episode_multi_version", ParentKey: contextParentKey, VariantKey: "variant:2160p", ReviewState: "auto"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}, ContextEvidence: map[uint][]ContextEvidence{1: {context}}})
	seenEpisode := false
	seenVariant := false
	for _, candidate := range output.Candidates {
		if candidate.CandidateType == CandidateTypeEpisode && candidate.CandidateKey == contextParentKey {
			seenEpisode = true
		}
		if candidate.CandidateType == CandidateTypeVariant && candidate.ParentCandidateKey == contextParentKey && candidate.VariantKey == context.VariantKey {
			seenVariant = true
		}
	}
	if !seenEpisode || !seenVariant {
		t.Fatalf("expected directory reduction to seed episode parent and variant, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryUsesDirectoryReductionParentForMovieCollection(t *testing.T) {
	year := 2024
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Movie.A.2024.mkv", Container: "mkv", ContentClass: "video", Status: "available"}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Movie A", Year: &year}
	parentKey := MovieWorkKey(MovieWorkInput{Title: "Movie A", Year: &year})
	context := ContextEvidence{Source: "directory_reduction", Assignment: "movie_collection", ParentKey: parentKey, TargetKey: "/library", ReviewState: "auto"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}, ContextEvidence: map[uint][]ContextEvidence{1: {context}}})
	seenWork := false
	for _, candidate := range output.Candidates {
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateKey == parentKey {
			seenWork = true
		}
	}
	if !seenWork {
		t.Fatalf("expected directory reduction to seed movie collection work candidate, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryUsesEpisodeIdentityContextToCreateSeriesSeasonEpisode(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 1/01.mkv", Container: "mkv", ContentClass: "video", Status: "available"}
	episode := 1
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "01", EpisodeNumber: &episode}
	context := []ContextEvidence{
		{Source: "content_shape", Assignment: "series_identity", ParentKey: SeriesWorkKey("Show"), Payload: map[string]any{"series_title": "Show"}, ReviewState: "auto"},
		{Source: "content_shape", Assignment: "season_identity", ParentKey: SeasonWorkKey("Show", 1), Payload: map[string]any{"series_title": "Show", "season_number": 1}, ReviewState: "auto"},
		{Source: "content_shape", Assignment: "episode_identity", ParentKey: EpisodeKey(EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 1}), Payload: map[string]any{"series_title": "Show", "season_number": 1, "episode_number": 1}, ReviewState: "auto"},
	}
	output := BuildManifestFromInventory(ManifestBuildInput{
		Scope:           ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library/Show/Season 1", ClassifierVersion: "test"},
		Files:           []database.InventoryFile{file},
		FileSignals:     map[uint]database.InventoryFileSignal{1: signal},
		ContextEvidence: map[uint][]ContextEvidence{1: context},
	})
	seenSeries := false
	seenSeason := false
	seenEpisode := false
	for _, candidate := range output.Candidates {
		switch {
		case candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindSeries && candidate.CandidateKey == SeriesWorkKey("Show"):
			seenSeries = true
		case candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindSeason && candidate.CandidateKey == SeasonWorkKey("Show", 1):
			seenSeason = true
		case candidate.CandidateType == CandidateTypeEpisode && candidate.CandidateKey == EpisodeKey(EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 1}):
			seenEpisode = true
		}
	}
	if !seenSeries || !seenSeason || !seenEpisode {
		t.Fatalf("expected series/season/episode candidates, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryParentsSingleEpisodeResourceToEpisode(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 1/01.mkv", StableIdentityKey: "ep-1", Container: "mkv", ContentClass: "video", Status: "available"}
	season := 1
	episode := 1
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: &episode}
	output := BuildManifestFromInventory(ManifestBuildInput{
		Scope:       ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library/Show/Season 1", ClassifierVersion: "test"},
		Files:       []database.InventoryFile{file},
		FileSignals: map[uint]database.InventoryFileSignal{1: signal},
	})
	resource, ok := candidateByType(output.Candidates, CandidateTypePlayableResource)
	if !ok {
		t.Fatalf("expected playable resource candidate, got %#v", output.Candidates)
	}
	wantEpisodeKey := EpisodeKey(EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 1})
	if resource.ParentCandidateKey != wantEpisodeKey || resource.CanonicalKey != wantEpisodeKey {
		t.Fatalf("expected resource parent/canonical key %q, got %#v", wantEpisodeKey, resource)
	}
}

func TestBuildManifestFromInventoryMarksMultiEpisodeResourceWithEpisodeKeys(t *testing.T) {
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show/Show.S01E01-E02.mkv", StableIdentityKey: "multi-ep", Container: "mkv", ContentClass: "video", Status: "available"}
	season := 1
	episode := 1
	signal := database.InventoryFileSignal{
		InventoryFileID:    &file.ID,
		TitleCandidate:     "Show",
		SeasonNumber:       &season,
		EpisodeNumber:      &episode,
		EpisodeNumbersJSON: `[1,2]`,
	}
	output := BuildManifestFromInventory(ManifestBuildInput{
		Scope:       ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library/Show", ClassifierVersion: "test"},
		Files:       []database.InventoryFile{file},
		FileSignals: map[uint]database.InventoryFileSignal{1: signal},
	})
	resource, ok := candidateByType(output.Candidates, CandidateTypePlayableResource)
	if !ok {
		t.Fatalf("expected playable resource candidate, got %#v", output.Candidates)
	}
	if resource.ResourceShape != ResourceKindMultiEpisode {
		t.Fatalf("expected multi-episode resource shape, got %#v", resource)
	}
	if !strings.Contains(resource.EvidenceJSON, `"episode_keys"`) {
		t.Fatalf("expected episode_keys in resource evidence, got %#v", resource)
	}
}

func TestBuildManifestFromInventoryDoesNotCreateEpisodeForMovieCollectionParent(t *testing.T) {
	episode := 1
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/1-cwdv-027-shiori-uehara-catwalk-poison-27_hq.mp4", Container: "mp4", ContentClass: "video", Status: "available"}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "1 cwdv 027 shiori uehara catwalk poison 27 hq", EpisodeNumber: &episode, EpisodeSource: "leading_numeric"}
	parentKey := MovieWorkKey(MovieWorkInput{Title: signal.TitleCandidate})
	context := ContextEvidence{Source: "directory_reduction", Assignment: "movie_collection", ParentKey: parentKey, TargetKey: "/library", ReviewState: "auto"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}, ContextEvidence: map[uint][]ContextEvidence{1: {context}}})

	seenWork := false
	for _, candidate := range output.Candidates {
		if candidate.CandidateKey != parentKey {
			continue
		}
		if candidate.CandidateType == CandidateTypeEpisode {
			t.Fatalf("did not expect movie collection parent to also create episode candidate, got %#v", output.Candidates)
		}
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie {
			seenWork = true
		}
	}
	if !seenWork {
		t.Fatalf("expected movie collection work candidate, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryUsesLeadingNumericTitleForMovieCollection(t *testing.T) {
	episode := 1
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/1-lafbd-70-hina-makimura-laforet-girl-70_hq.mp4", Container: "mp4", ContentClass: "video", Status: "available"}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "1 lafbd 70 hina makimura laforet girl 70 hq", EpisodeNumber: &episode, EpisodeSource: "leading_numeric"}
	parentKey := MovieWorkKey(MovieWorkInput{Title: "1 lafbd 70 hina makimura laforet girl 70 hq"})
	context := ContextEvidence{Source: "directory_reduction", Assignment: "movie_collection", ParentKey: parentKey, TargetKey: "/library", ReviewState: "auto"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}, ContextEvidence: map[uint][]ContextEvidence{1: {context}}})
	seenWork := false
	for _, candidate := range output.Candidates {
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateKey == parentKey {
			seenWork = true
		}
	}
	if !seenWork {
		t.Fatalf("expected leading numeric movie collection file to create work candidate, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryPreservesMovieCollectionParentKey(t *testing.T) {
	episode := 1
	files := []database.InventoryFile{
		{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/1-cwdv-027-shiori-uehara-catwalk-poison-27_hq.mp4", Container: "mp4", ContentClass: "video", Status: "available"},
		{ID: 2, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/1-cwdv-028-ryoko-murakami-catwalk-poison-28_hq.mp4", Container: "mp4", ContentClass: "video", Status: "available"},
	}
	signals := map[uint]database.InventoryFileSignal{
		1: {InventoryFileID: &files[0].ID, TitleCandidate: "1 cwdv 027 shiori uehara catwalk poison 27 hq", EpisodeNumber: &episode, EpisodeSource: "leading_numeric"},
		2: {InventoryFileID: &files[1].ID, TitleCandidate: "1 cwdv 028 ryoko murakami catwalk poison 28 hq", EpisodeNumber: &episode, EpisodeSource: "leading_numeric"},
	}
	firstKey := MovieWorkKey(MovieWorkInput{Title: signals[1].TitleCandidate})
	secondKey := MovieWorkKey(MovieWorkInput{Title: signals[2].TitleCandidate})
	context := map[uint][]ContextEvidence{
		1: {{Source: "directory_reduction", Assignment: "movie_collection", ParentKey: firstKey, TargetKey: "/library", ReviewState: "auto"}},
		2: {{Source: "directory_reduction", Assignment: "movie_collection", ParentKey: secondKey, TargetKey: "/library", ReviewState: "auto"}},
	}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: files, FileSignals: signals, ContextEvidence: context})
	seen := map[string]bool{}
	for _, candidate := range output.Candidates {
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie {
			seen[candidate.CandidateKey] = true
		}
	}
	if !seen[firstKey] || !seen[secondKey] {
		t.Fatalf("expected distinct directory parent keys to survive title cleaning, got %#v", output.Candidates)
	}
}

func TestBuildManifestFromInventoryRejectsSeasonEpisodeTitleForMovie(t *testing.T) {
	season := 1
	episode := 2
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/Show.S01E02.mkv", Container: "mkv", ContentClass: "video", Status: "available"}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "Show", SeasonNumber: &season, EpisodeNumber: &episode, EpisodeSource: "explicit"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}})
	for _, candidate := range output.Candidates {
		if candidate.CandidateType == CandidateTypeWork && candidate.CandidateRole == WorkKindMovie {
			t.Fatalf("did not expect explicit season episode to create movie work candidate, got %#v", output.Candidates)
		}
	}
}

func TestBuildManifestFromInventoryUsesSharedStrongMovieNormalization(t *testing.T) {
	year := 2002
	file := database.InventoryFile{ID: 1, LibraryID: 1, StorageProvider: "local", StoragePath: "/library/28.Days.Later.2002.BluRay.1080p.DTS-HD.MA5.1.x265.10bit-Xiaomi.mkv", Container: "mkv", ContentClass: "video", Status: "available"}
	signal := database.InventoryFileSignal{InventoryFileID: &file.ID, TitleCandidate: "28 Days Later MA5 1", Year: &year, Audio: "DTS-HD.MA5.1", Quality: "1080p", Codec: "x265"}
	output := BuildManifestFromInventory(ManifestBuildInput{Scope: ManifestScope{LibraryID: 1, StorageProvider: "local", RootPath: "/library", ScopePath: "/library", ClassifierVersion: "test"}, Files: []database.InventoryFile{file}, FileSignals: map[uint]database.InventoryFileSignal{1: signal}})
	work, ok := candidateByType(output.Candidates, CandidateTypeWork)
	if !ok {
		t.Fatalf("expected work candidate, got %#v", output.Candidates)
	}
	if work.CandidateKey != "work:movie:28-days-later:2002" {
		t.Fatalf("expected shared strong movie normalization key, got %#v", work)
	}
}

func candidateTypeCounts(candidates []database.RecognitionCandidate) map[string]int {
	counts := make(map[string]int)
	for _, candidate := range candidates {
		counts[candidate.CandidateType]++
	}
	return counts
}
