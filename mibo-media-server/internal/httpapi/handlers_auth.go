package httpapi

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/catalog"
	"github.com/atlan/mibo-media-server/internal/progress"
)

func (r *Router) handleRegister(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	user, err := r.auth.Register(req.Context(), input.Username, input.Password)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusCreated, user)
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	result, err := r.auth.Login(req.Context(), input.Username, input.Password, loginMetadataFromRequest(req))
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	writeJSON(req.Context(), w, http.StatusOK, result)
}

func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	token, err := bearerToken(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if err := r.auth.Logout(req.Context(), token); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (r *Router) handleListAuthSessions(w http.ResponseWriter, req *http.Request) {
	token, err := bearerToken(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	user, err := r.auth.Authenticate(req.Context(), token)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	sessions, err := r.auth.ListLoginSessions(req.Context(), user.ID, token)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, sessions)
}

func (r *Router) handleRevokeAuthSession(w http.ResponseWriter, req *http.Request) {
	token, err := bearerToken(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	user, err := r.auth.Authenticate(req.Context(), token)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	sessionID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if err := r.auth.RevokeLoginSession(req.Context(), user.ID, sessionID, token); err != nil {
		switch {
		case errors.Is(err, auth.ErrCurrentSession):
			writeError(req.Context(), w, http.StatusBadRequest, err)
		case errors.Is(err, auth.ErrSessionNotFound):
			writeError(req.Context(), w, http.StatusNotFound, err)
		default:
			writeError(req.Context(), w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"id": sessionID, "status": "revoked"})
}

func (r *Router) handleRevokeOtherAuthSessions(w http.ResponseWriter, req *http.Request) {
	token, err := bearerToken(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	user, err := r.auth.Authenticate(req.Context(), token)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if err := r.auth.RevokeOtherLoginSessions(req.Context(), user.ID, token); err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, map[string]any{"status": "revoked"})
}

func (r *Router) handleMe(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, user)
}

func loginMetadataFromRequest(req *http.Request) auth.LoginMetadata {
	userAgent := strings.TrimSpace(req.UserAgent())
	return auth.LoginMetadata{
		UserAgent:  userAgent,
		RemoteAddr: requestRemoteAddr(req),
		DeviceName: deviceNameFromUserAgent(userAgent),
		ClientType: clientTypeFromUserAgent(userAgent),
	}
}

func requestRemoteAddr(req *http.Request) string {
	remoteAddr := strings.TrimSpace(req.RemoteAddr)
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func deviceNameFromUserAgent(userAgent string) string {
	lower := strings.ToLower(userAgent)
	switch {
	case userAgent == "":
		return ""
	case strings.Contains(lower, "iphone"):
		return "iPhone"
	case strings.Contains(lower, "ipad"):
		return "iPad"
	case strings.Contains(lower, "android"):
		return "Android device"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os"):
		return "Mac"
	case strings.Contains(lower, "windows"):
		return "Windows PC"
	case strings.Contains(lower, "linux"):
		return "Linux device"
	default:
		return "Unknown device"
	}
}

func clientTypeFromUserAgent(userAgent string) string {
	if strings.TrimSpace(userAgent) == "" {
		return ""
	}
	return "Mibo Web"
}

func (r *Router) handleUpdateProgress(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	var input progress.UpdateInput
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	frameURL, err := r.saveProgressFrame(user.ID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if frameURL != "" {
		input.ProgressFrameURL = frameURL
	}

	state, err := r.progress.Update(req.Context(), user.ID, input)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, state)
}

func (r *Router) saveProgressFrame(userID uint, input progress.UpdateInput) (string, error) {
	frame := strings.TrimSpace(input.ProgressFrameData)
	if frame == "" {
		return "", nil
	}
	if input.ItemID == 0 {
		return "", fmt.Errorf("item_id is required for progress frame")
	}

	mediaType, payload, ok := strings.Cut(frame, ",")
	if !ok || !strings.HasPrefix(mediaType, "data:image/") || !strings.Contains(mediaType, ";base64") {
		return "", fmt.Errorf("progress_frame_data must be a base64 image data URL")
	}
	ext := progressFrameExtension(mediaType)
	if ext == "" {
		return "", fmt.Errorf("progress_frame_data must be jpeg, png, or webp")
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("decode progress frame: %w", err)
	}
	if len(decoded) == 0 || len(decoded) > 2*1024*1024 {
		return "", fmt.Errorf("progress frame must be between 1 byte and 2 MiB")
	}

	dir := filepath.Join(r.generatedArtworkRootPath(), "progress", strconvFormatUint(userID), strconvFormatUint(input.ItemID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	baseName := "default"
	if input.AssetID != nil && *input.AssetID > 0 {
		baseName = "asset-" + strconvFormatUint(*input.AssetID)
	}
	for _, staleExt := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		_ = os.Remove(filepath.Join(dir, baseName+staleExt))
	}
	path := filepath.Join(dir, baseName+ext)
	if err := os.WriteFile(path, decoded, 0o644); err != nil {
		return "", err
	}

	return "/api/v1/me/progress-frames/" + strconvFormatUint(input.ItemID) + "/" + baseName, nil
}

func progressFrameExtension(mediaType string) string {
	switch {
	case strings.HasPrefix(mediaType, "data:image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(mediaType, "data:image/png"):
		return ".png"
	case strings.HasPrefix(mediaType, "data:image/webp"):
		return ".webp"
	default:
		return ""
	}
}

func (r *Router) handleContinueWatching(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.catalog.ListContinueWatching(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeUserItemEntryURLs(req, entries)
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) handleListFavorites(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.catalog.ListFavorites(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func (r *Router) handleAddFavorite(w http.ResponseWriter, req *http.Request) {
	r.handleFavoriteMutation(w, req, true)
}

func (r *Router) handleRemoveFavorite(w http.ResponseWriter, req *http.Request) {
	r.handleFavoriteMutation(w, req, false)
}

func (r *Router) handleFavoriteMutation(w http.ResponseWriter, req *http.Request, favorite bool) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	itemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	entry, err := r.catalog.SetFavorite(req.Context(), user.ID, itemID, favorite)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, entry)
}

func (r *Router) handleRecentlyPlayed(w http.ResponseWriter, req *http.Request) {
	user, err := r.requireUser(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	entries, err := r.catalog.ListRecentlyPlayed(req.Context(), user.ID, limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeUserItemEntryURLs(req, entries)
	writeJSON(req.Context(), w, http.StatusOK, entries)
}

func normalizeUserItemEntryURLs(req *http.Request, entries []catalog.CatalogUserItemEntry) {
	for idx := range entries {
		entries[idx].ProgressFrameURL = buildAssetURL(req, entries[idx].ProgressFrameURL)
		normalizeCatalogListItemArtworkURLs(req, &entries[idx].Item)
		if entries[idx].DisplayItem != nil {
			normalizeCatalogListItemArtworkURLs(req, entries[idx].DisplayItem)
		}
		if entries[idx].PlayItem != nil {
			normalizeCatalogListItemArtworkURLs(req, entries[idx].PlayItem)
		}
	}
}

func (r *Router) handleRecentlyAdded(w http.ResponseWriter, req *http.Request) {
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	items, err := r.catalog.ListRecentlyAdded(req.Context(), limit)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	normalizeCatalogListItemsArtworkURLs(req, items)
	writeJSON(req.Context(), w, http.StatusOK, items)
}

func (r *Router) handleLatestByLibrary(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	if r.catalog == nil {
		writeError(req.Context(), w, http.StatusInternalServerError, errors.New("catalog service unavailable"))
		return
	}
	sections, err := r.catalog.ListLatestByLibrary(req.Context(), 12)
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	for idx := range sections {
		normalizeCatalogListItemsArtworkURLs(req, sections[idx].Items)
	}

	writeJSON(req.Context(), w, http.StatusOK, sections)
}
