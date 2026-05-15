package scanrecognition

import (
	"encoding/xml"
	"strconv"
	"strings"
)

const maxNFOTextBytes = 1 << 20

type NFOSignal struct {
	Kind            DirectoryKind
	TitleCandidates []string
	Year            *int
	Season          *int
	Episode         *int
	ExternalIDs     map[string]string
}

type nfoDocument struct {
	XMLName xml.Name
	Title   string `xml:"title"`
	Year    string `xml:"year"`
	Season  string `xml:"season"`
	Episode string `xml:"episode"`
	TMDBID  string `xml:"tmdbid"`
	IMDbID  string `xml:"imdbid"`
}

func ParseNFO(input string) NFOSignal {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || len(trimmed) > maxNFOTextBytes {
		return NFOSignal{Kind: DirectoryKindUnknown}
	}
	var document nfoDocument
	if err := xml.Unmarshal([]byte(trimmed), &document); err != nil {
		return NFOSignal{Kind: DirectoryKindUnknown}
	}
	kind := nfoKind(document.XMLName.Local)
	if kind == DirectoryKindUnknown {
		return NFOSignal{Kind: DirectoryKindUnknown}
	}

	signal := NFOSignal{
		Kind:            kind,
		TitleCandidates: titleCandidates(document.Title),
		Year:            parseOptionalInt(document.Year),
		Season:          parseOptionalInt(document.Season),
		Episode:         parseOptionalInt(document.Episode),
		ExternalIDs:     nfoExternalIDs(document),
	}
	return signal
}

func nfoKind(rootName string) DirectoryKind {
	switch strings.ToLower(strings.TrimSpace(rootName)) {
	case "movie":
		return DirectoryKindMovie
	case "tvshow", "series":
		return DirectoryKindSeries
	case "season":
		return DirectoryKindSeason
	case "episodedetails", "episode":
		return DirectoryKindEpisodeGroup
	default:
		return DirectoryKindUnknown
	}
}

func parseOptionalInt(input string) *int {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil
	}
	return &value
}

func nfoExternalIDs(document nfoDocument) map[string]string {
	ids := map[string]string{}
	if value := strings.TrimSpace(document.TMDBID); value != "" {
		ids["tmdb"] = value
	}
	if value := strings.TrimSpace(document.IMDbID); value != "" {
		ids["imdb"] = value
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}
