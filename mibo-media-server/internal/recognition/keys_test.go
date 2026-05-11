package recognition

import "testing"

func TestMovieWorkKeyIncludesNormalizedTitleAndYear(t *testing.T) {
	year := 2004
	key := MovieWorkKey(MovieWorkInput{Title: "3.Iron", Year: &year})
	if key != "work:movie:3-iron:2004" {
		t.Fatalf("unexpected movie work key %q", key)
	}
}

func TestEpisodeKeyUsesSeriesSeasonEpisodeTuple(t *testing.T) {
	key := EpisodeKey(EpisodeInput{SeriesTitle: "Show Name", SeasonNumber: 1, EpisodeNumber: 2})
	if key != "episode:work:season:work:series:show-name:s01:e02" {
		t.Fatalf("unexpected episode key %q", key)
	}
}

func TestPlayableResourceKeyPrefersStableIdentity(t *testing.T) {
	key := PlayableResourceKey(ResourceInput{StorageProvider: "local", StoragePath: "/library/Movie.mkv", StableIdentityKey: "file-123"})
	if key != "playable_resource:local:stable:file-123" {
		t.Fatalf("unexpected resource key %q", key)
	}
}

func TestVariantKeyIsOrderIndependent(t *testing.T) {
	left := VariantKey(VariantInput{Quality: "2160p", SourceTags: []string{"UHD", "BluRay"}, Codec: "x265"})
	right := VariantKey(VariantInput{Codec: "x265", SourceTags: []string{"BluRay", "UHD"}, Quality: "2160p"})
	if left == "" || left != right {
		t.Fatalf("expected stable variant keys, got %q and %q", left, right)
	}
}
