package scanrecognition

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseNFOExtractsMovieSignal(t *testing.T) {
	got := ParseNFO(`<movie><title>Inception</title><year>2010</year><tmdbid>27205</tmdbid></movie>`)
	want := NFOSignal{
		Kind:            DirectoryKindMovie,
		TitleCandidates: []string{"Inception"},
		Year:            intPtr(2010),
		ExternalIDs:     map[string]string{"tmdb": "27205"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestParseNFOExtractsSeriesSignal(t *testing.T) {
	got := ParseNFO(`<tvshow><title>Six Feet Under</title><year>2001</year><imdbid>tt0248654</imdbid></tvshow>`)
	want := NFOSignal{
		Kind:            DirectoryKindSeries,
		TitleCandidates: []string{"Six Feet Under"},
		Year:            intPtr(2001),
		ExternalIDs:     map[string]string{"imdb": "tt0248654"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestParseNFOReturnsUnknownForInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "malformed", input: `<movie><title>Broken`},
		{name: "empty", input: ``},
		{name: "unknown root", input: `<artist><title>Someone</title></artist>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseNFO(tt.input)
			if got.Kind != DirectoryKindUnknown {
				t.Fatalf("expected unknown NFO, got %#v", got)
			}
		})
	}
}

func TestParseNFORejectsOversizedInput(t *testing.T) {
	got := ParseNFO(strings.Repeat("x", maxNFOTextBytes+1))
	if got.Kind != DirectoryKindUnknown {
		t.Fatalf("expected oversized NFO to return unknown, got %#v", got)
	}
}

func TestParseNFOExtractsEpisodeSignal(t *testing.T) {
	got := ParseNFO(`<episodedetails><title>Pilot</title><season>1</season><episode>2</episode></episodedetails>`)
	want := NFOSignal{
		Kind:            DirectoryKindEpisodeGroup,
		TitleCandidates: []string{"Pilot"},
		Season:          intPtr(1),
		Episode:         intPtr(2),
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
