package httpapi

import (
	"net/http"

	"github.com/atlan/mibo-media-server/internal/database"
)

func (r *Router) handleSetupStatus(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var userCount int64
	if err := r.db.WithContext(ctx).Model(&database.User{}).Count(&userCount).Error; err != nil {
		writeError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	var mediaSourceCount int64
	if err := r.db.WithContext(ctx).Model(&database.MediaSource{}).Count(&mediaSourceCount).Error; err != nil {
		writeError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	var libraryCount int64
	if err := r.db.WithContext(ctx).Model(&database.Library{}).Count(&libraryCount).Error; err != nil {
		writeError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(ctx, w, http.StatusOK, map[string]any{
		"initialized":        userCount > 0 && mediaSourceCount > 0 && libraryCount > 0,
		"can_enter_app":      userCount > 0,
		"has_users":          userCount > 0,
		"has_media_sources":  mediaSourceCount > 0,
		"has_libraries":      libraryCount > 0,
		"user_count":         userCount,
		"media_source_count": mediaSourceCount,
		"library_count":      libraryCount,
	})
}
