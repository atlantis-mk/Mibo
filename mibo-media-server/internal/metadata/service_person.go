package metadata

import (
	"context"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (s *Service) RefreshCatalogPersonProfile(ctx context.Context, personID uint) error {
	var person database.Person
	if err := s.db.WithContext(ctx).Where("id = ?", personID).First(&person).Error; err != nil {
		return err
	}
	if person.TMDBPersonID == nil || *person.TMDBPersonID <= 0 {
		return nil
	}

	tmdbCfg, err := s.tmdbConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tmdbCfg.APIKey) == "" {
		return nil
	}

	profile, err := s.fetchPersonDetail(ctx, tmdbCfg, *person.TMDBPersonID)
	if err != nil {
		return err
	}

	refreshedAt := time.Now().UTC()
	updates := map[string]any{
		"avatar_url":           strings.TrimSpace(firstNonEmpty(imageURL(tmdbCfg, profile.ProfilePath), person.AvatarURL)),
		"imdb_id":              strings.TrimSpace(profile.IMDbID),
		"biography":            strings.TrimSpace(profile.Biography),
		"birthday":             parseProviderDate(profile.Birthday),
		"deathday":             parseProviderDate(profile.Deathday),
		"place_of_birth":       strings.TrimSpace(profile.PlaceOfBirth),
		"known_for_department": strings.TrimSpace(profile.KnownForDepartment),
		"profile_refreshed_at": &refreshedAt,
	}
	return s.db.WithContext(ctx).Model(&database.Person{}).Where("id = ?", personID).Updates(updates).Error
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
