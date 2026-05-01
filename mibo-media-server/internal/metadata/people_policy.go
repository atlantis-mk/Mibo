package metadata

import (
	"context"
	"strings"

	"github.com/atlan/mibo-media-server/internal/library"
)

func (s *Service) applyNormalizedPeople(ctx context.Context, itemID uint, people []NormalizedMetadataPerson, sourceID *uint) error {
	cast := make([]library.PersonDetail, 0)
	directors := make([]library.PersonDetail, 0)
	for _, person := range people {
		name := strings.TrimSpace(person.Name)
		if name == "" {
			continue
		}
		detail := library.PersonDetail{Name: name, Role: strings.TrimSpace(firstNonEmpty(person.Character, person.Role)), AvatarURL: strings.TrimSpace(person.AvatarURL), TMDBPersonID: person.TMDBPersonID}
		switch strings.TrimSpace(person.Role) {
		case "director":
			directors = append(directors, detail)
		default:
			cast = append(cast, detail)
		}
	}
	return s.syncCatalogPeople(ctx, itemID, cast, directors, sourceID)
}
