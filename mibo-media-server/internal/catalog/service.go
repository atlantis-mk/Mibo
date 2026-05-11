package catalog

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/ingest"
	"gorm.io/gorm"
)

const (
	ItemTypeMovie      = "movie"
	ItemTypeSeries     = "series"
	ItemTypeSeason     = "season"
	ItemTypeEpisode    = "episode"
	ItemTypeExtra      = "extra"
	ItemTypeCollection = "collection"

	AvailabilityAvailable    = "available"
	AvailabilityMissing      = "missing"
	AvailabilityUnaired      = "unaired"
	AvailabilityNoLocalMedia = "no_local_media"

	GovernancePending     = "pending"
	GovernanceMatched     = "matched"
	GovernanceNeedsReview = "needs_review"
	GovernanceLocked      = "locked"
	GovernanceManual      = "manual"
	GovernanceUnmatched   = "unmatched"

	SourceTypeProvider  = "provider"
	SourceTypeLocalFile = "local_file"
	SourceTypeManual    = "manual"
	SourceTypeNFO       = "nfo"
)

type Service struct {
	db                     *gorm.DB
	personProfileRefresher PersonProfileRefresher
	ingest                 *ingest.Service
}

type PersonProfileRefresher interface {
	RefreshCatalogPersonProfile(ctx context.Context, personID uint) error
}

func NewService(db *gorm.DB, args ...any) *Service {
	service := &Service{db: db}
	for _, arg := range args {
		if ingestSvc, ok := arg.(*ingest.Service); ok {
			service.ingest = ingestSvc
		}
	}
	return service
}

func (s *Service) SetPersonProfileRefresher(refresher PersonProfileRefresher) {
	s.personProfileRefresher = refresher
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
