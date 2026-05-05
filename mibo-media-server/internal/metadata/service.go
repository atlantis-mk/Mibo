package metadata

import (
	"context"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/ingest"
	"github.com/atlan/mibo-media-server/internal/search"
	"github.com/atlan/mibo-media-server/internal/settings"
	"gorm.io/gorm"
)

const (
	StatusPending     = "pending"
	StatusMatched     = "matched"
	StatusNeedsReview = "needs_review"
	StatusUnmatched   = "unmatched"
	StatusSkipped     = "skipped"
)

type Service struct {
	db       *gorm.DB
	fallback config.MetadataConfig
	settings *settings.Service
	search   *search.Service
	ingest   *ingest.Service
}

type ManualSearchInput struct {
	Title  string `json:"title"`
	Year   *int   `json:"year"`
	IMDbID string `json:"imdb_id"`
	TMDBID string `json:"tmdb_id"`
	TVDBID string `json:"tvdb_id"`
}

type ApplyCandidateInput struct {
	ExternalID string `json:"external_id"`
}

type ManualMetadataInput struct {
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	Year          *int   `json:"year"`
	Overview      string `json:"overview"`
	PosterURL     string `json:"poster_url"`
	BackdropURL   string `json:"backdrop_url"`
}

type SearchCandidate struct {
	Provider      string  `json:"provider"`
	MediaType     string  `json:"media_type"`
	ExternalID    string  `json:"external_id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	PosterURL     string  `json:"poster_url"`
	BackdropURL   string  `json:"backdrop_url"`
	ReleaseDate   string  `json:"release_date"`
	Year          *int    `json:"year,omitempty"`
	Confidence    float64 `json:"confidence"`
	MatchedQuery  string  `json:"matched_query,omitempty"`
	ReasonSummary string  `json:"reason_summary,omitempty"`
}

type tmdbRequestFailure struct {
	statusCode int
	message    string
}

func (e tmdbRequestFailure) Error() string {
	return e.message
}

func (e tmdbRequestFailure) StatusCode() int {
	return e.statusCode
}

type tmdbErrorResponse struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Success       bool   `json:"success"`
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type searchResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	OriginalTitle string  `json:"original_title"`
	OriginalName  string  `json:"original_name"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	Popularity    float64 `json:"popularity"`
	VoteCount     int     `json:"vote_count"`
}

type detailResponse struct {
	ID                  int                    `json:"id"`
	Runtime             *int                   `json:"runtime"`
	EpisodeRunTime      []int                  `json:"episode_run_time"`
	Title               string                 `json:"title"`
	Name                string                 `json:"name"`
	OriginalTitle       string                 `json:"original_title"`
	OriginalName        string                 `json:"original_name"`
	Overview            string                 `json:"overview"`
	PosterPath          string                 `json:"poster_path"`
	BackdropPath        string                 `json:"backdrop_path"`
	ReleaseDate         string                 `json:"release_date"`
	FirstAirDate        string                 `json:"first_air_date"`
	LastAirDate         string                 `json:"last_air_date"`
	Status              string                 `json:"status"`
	Genres              []namedValue           `json:"genres"`
	ProductionCountries []countryValue         `json:"production_countries"`
	Seasons             []seasonSummary        `json:"seasons"`
	CreatedBy           []namedValue           `json:"created_by"`
	Credits             creditsResponse        `json:"credits"`
	Images              imagesResponse         `json:"images"`
	Videos              videosResponse         `json:"videos"`
	Keywords            keywordsResponse       `json:"keywords"`
	ReleaseDates        releaseDatesResponse   `json:"release_dates"`
	ContentRatings      contentRatingsResponse `json:"content_ratings"`
	ExternalIDs         externalIDsResponse    `json:"external_ids"`
	VoteAverage         float64                `json:"vote_average"`
}

type keywordsResponse struct {
	Keywords []namedValue `json:"keywords"`
	Results  []namedValue `json:"results"`
}

type releaseDatesResponse struct {
	Results []releaseDateRegion `json:"results"`
}

type releaseDateRegion struct {
	Region       string                     `json:"iso_3166_1"`
	ReleaseDates []releaseDateCertification `json:"release_dates"`
}

type releaseDateCertification struct {
	Certification string `json:"certification"`
}

type contentRatingsResponse struct {
	Results []contentRating `json:"results"`
}

type contentRating struct {
	Region string `json:"iso_3166_1"`
	Rating string `json:"rating"`
}

type externalIDsResponse struct {
	IMDbID     string `json:"imdb_id"`
	TVDBID     int    `json:"tvdb_id"`
	WikidataID string `json:"wikidata_id"`
}

type personDetailResponse struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Biography          string `json:"biography"`
	Birthday           string `json:"birthday"`
	Deathday           string `json:"deathday"`
	PlaceOfBirth       string `json:"place_of_birth"`
	KnownForDepartment string `json:"known_for_department"`
	ProfilePath        string `json:"profile_path"`
	IMDbID             string `json:"imdb_id"`
}

type videosResponse struct {
	Results []videoAsset `json:"results"`
}

type videoAsset struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Site      string `json:"site"`
	Type      string `json:"type"`
	Official  bool   `json:"official"`
	Language  string `json:"iso_639_1"`
	Published string `json:"published_at"`
}

type seasonSummary struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
}

type seasonDetailResponse struct {
	ID           int                     `json:"id"`
	SeasonNumber int                     `json:"season_number"`
	Name         string                  `json:"name"`
	AirDate      string                  `json:"air_date"`
	Overview     string                  `json:"overview"`
	PosterPath   string                  `json:"poster_path"`
	Credits      creditsResponse         `json:"credits"`
	ExternalIDs  externalIDsResponse     `json:"external_ids"`
	Episodes     []seasonEpisodeResponse `json:"episodes"`
}

type seasonEpisodeResponse struct {
	ID            int          `json:"id"`
	SeasonNumber  int          `json:"season_number"`
	EpisodeNumber int          `json:"episode_number"`
	Name          string       `json:"name"`
	AirDate       string       `json:"air_date"`
	Overview      string       `json:"overview"`
	StillPath     string       `json:"still_path"`
	Runtime       *int         `json:"runtime"`
	VoteAverage   float64      `json:"vote_average"`
	Crew          []crewMember `json:"crew"`
	GuestStars    []castMember `json:"guest_stars"`
}

type imagesResponse struct {
	Logos []imageAsset `json:"logos"`
}

type imageAsset struct {
	FilePath    string  `json:"file_path"`
	Language    string  `json:"iso_639_1"`
	VoteAverage float64 `json:"vote_average"`
}

type namedValue struct {
	Name string `json:"name"`
}

type countryValue struct {
	Name string `json:"name"`
}

type creditsResponse struct {
	Cast []castMember `json:"cast"`
	Crew []crewMember `json:"crew"`
}

type castMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	ProfilePath string `json:"profile_path"`
}

type crewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}

func NewService(db *gorm.DB, cfg config.MetadataConfig, settingsSvc *settings.Service, args ...any) *Service {
	service := &Service{db: db, fallback: cfg, settings: settingsSvc}
	for _, arg := range args {
		if searchSvc, ok := arg.(*search.Service); ok {
			service.search = searchSvc
		}
		if ingestSvc, ok := arg.(*ingest.Service); ok {
			service.ingest = ingestSvc
		}
	}
	return service
}

func (s *Service) tmdbConfig(ctx context.Context) (config.TMDBConfig, error) {
	if s.settings == nil {
		return s.fallback.TMDB, nil
	}
	resolved, _, err := s.settings.ResolveTMDBConfig(ctx)
	if err != nil {
		return config.TMDBConfig{}, err
	}
	return resolved, nil
}
