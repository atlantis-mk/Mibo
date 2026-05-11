package catalog

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
	"gorm.io/gorm"
)

const personProfileRefreshTTL = 7 * 24 * time.Hour

func (s *Service) GetPersonDetail(ctx context.Context, personID uint) (CatalogPersonPageDetail, error) {
	person, err := s.loadCatalogPerson(ctx, personID)
	if err != nil {
		return CatalogPersonPageDetail{}, err
	}

	if s.personProfileRefresher != nil && personProfileNeedsRefresh(person) {
		if refreshErr := s.personProfileRefresher.RefreshCatalogPersonProfile(ctx, personID); refreshErr == nil {
			if refreshed, err := s.loadCatalogPerson(ctx, personID); err == nil {
				person = refreshed
			}
		}
	}

	relatedItems, err := s.loadPersonRelatedCatalogItems(ctx, person.ID, 24)
	if err != nil {
		return CatalogPersonPageDetail{}, err
	}

	return CatalogPersonPageDetail{
		ID:                 person.ID,
		Name:               strings.TrimSpace(person.Name),
		SortName:           strings.TrimSpace(person.SortName),
		AvatarURL:          strings.TrimSpace(person.AvatarURL),
		Biography:          strings.TrimSpace(person.Biography),
		Birthday:           person.Birthday,
		Deathday:           person.Deathday,
		PlaceOfBirth:       strings.TrimSpace(person.PlaceOfBirth),
		KnownForDepartment: strings.TrimSpace(person.KnownForDepartment),
		ExternalIdentities: ensureCatalogExternalIdentities(buildCatalogPersonExternalIdentities(person)),
		RelatedItems:       ensureCatalogListItems(relatedItems),
	}, nil
}

func (s *Service) loadCatalogPerson(ctx context.Context, personID uint) (database.Person, error) {
	var person database.Person
	err := s.db.WithContext(ctx).Where("id = ?", personID).First(&person).Error
	return person, err
}

func personProfileNeedsRefresh(person database.Person) bool {
	if person.TMDBPersonID == nil || *person.TMDBPersonID <= 0 {
		return false
	}
	if person.ProfileRefreshedAt == nil || person.ProfileRefreshedAt.IsZero() {
		return true
	}
	if time.Since(*person.ProfileRefreshedAt) > personProfileRefreshTTL {
		return true
	}
	return strings.TrimSpace(person.Biography) == "" ||
		strings.TrimSpace(person.IMDBID) == "" ||
		person.Birthday == nil ||
		strings.TrimSpace(person.PlaceOfBirth) == "" ||
		strings.TrimSpace(person.KnownForDepartment) == ""
}

func buildCatalogPersonExternalIdentities(person database.Person) []CatalogExternalIdentity {
	identities := make([]CatalogExternalIdentity, 0, 2)
	if person.TMDBPersonID != nil && *person.TMDBPersonID > 0 {
		identities = append(identities, CatalogExternalIdentity{
			Provider:     "tmdb",
			ProviderType: "person",
			ExternalID:   strings.TrimSpace(intToString(*person.TMDBPersonID)),
			IsPrimary:    true,
		})
	}
	if imdbID := strings.TrimSpace(person.IMDBID); imdbID != "" {
		identities = append(identities, CatalogExternalIdentity{
			Provider:     "imdb",
			ProviderType: "person",
			ExternalID:   imdbID,
			IsPrimary:    true,
		})
	}
	return identities
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func (s *Service) loadPersonRelatedCatalogItems(ctx context.Context, personID uint, limit int) ([]CatalogListItem, error) {
	_ = ctx
	_ = personID
	_ = limit
	return []CatalogListItem{}, nil
}

func IsPersonNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
