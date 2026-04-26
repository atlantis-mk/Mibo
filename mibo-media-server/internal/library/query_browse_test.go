package library

import (
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestGroupShowBrowseCandidatesPreserveSeasonSuffixWithoutMetadata(t *testing.T) {
	t.Parallel()

	groups := groupShowBrowseCandidates([]browseCandidate{
		{Item: database.MediaItem{ID: 2, LibraryID: 7, Type: "episode", Title: "灵笼 第一季 S01E01", SeriesTitle: "灵笼 第一季", SeasonNumber: intPtr(1), EpisodeNumber: intPtr(1)}, WatchRank: 1},
		{Item: database.MediaItem{ID: 3, LibraryID: 7, Type: "episode", Title: "灵笼 第二季 S02E01", SeriesTitle: "灵笼 第二季", SeasonNumber: intPtr(2), EpisodeNumber: intPtr(1)}, WatchRank: 0},
	})

	if len(groups) != 2 {
		t.Fatalf("expected two grouped shows, got %#v", groups)
	}
	if groups[0].Display.Title != "灵笼 第一季" || groups[0].Display.SeriesTitle != "灵笼 第一季" {
		t.Fatalf("expected first grouped title, got %#v", groups[0].Display)
	}
	if groups[1].Display.Title != "灵笼 第二季" || groups[1].Display.SeriesTitle != "灵笼 第二季" {
		t.Fatalf("expected second grouped title, got %#v", groups[1].Display)
	}
	if groups[0].Display.Type != string(BrowseTypeFilterShow) {
		t.Fatalf("expected grouped show display type, got %#v", groups[0].Display.Type)
	}
	if groups[1].Display.Type != string(BrowseTypeFilterShow) {
		t.Fatalf("expected grouped show display type, got %#v", groups[1].Display.Type)
	}
}

func intPtr(value int) *int {
	return &value
}
