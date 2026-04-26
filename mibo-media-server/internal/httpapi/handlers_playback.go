package httpapi

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/storage"
)

func (r *Router) handleGetPlaybackSource(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/items/"+req.PathValue("id")+"/playback") {
		return
	}
	if _, err := r.requireUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}

	mediaItemID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	preferredFileID, err := parseOptionalUintQuery(req, "file_id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	clientProfile, err := parseClientProfileQuery(req)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	source, err := r.playback.GetPlaybackSource(req.Context(), playback.PlaybackRequest{
		MediaItemID:      mediaItemID,
		PreferredFileID:  preferredFileID,
		ClientProfile:    clientProfile,
		AllowHLSFallback: r.hls.Enabled(),
	})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	source.URL = buildPlaybackURL(req, source.URL)

	writeJSON(req.Context(), w, http.StatusOK, source)
}

func parseClientProfileQuery(req *http.Request) (playback.ClientProfile, error) {
	value := playback.ClientProfile(strings.ToLower(strings.TrimSpace(req.URL.Query().Get("client_profile"))))
	switch value {
	case playback.ClientProfileWeb, playback.ClientProfileMobile, playback.ClientProfileTV:
		return value, nil
	case "":
		return "", fmt.Errorf("client_profile is required")
	default:
		return "", fmt.Errorf("client_profile must be one of web, mobile, tv")
	}
}

func (r *Router) handleGetMediaFileLink(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/assets/{id}/link") {
		return
	}
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	link, err := r.playback.GetFileLink(req.Context(), mediaFileID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	link.URL = buildPlaybackURL(req, link.URL)

	writeJSON(req.Context(), w, http.StatusOK, link)
}

func (r *Router) handleGetHLSPlaylist(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/inventory-files/{id}/hls/index.m3u8") {
		return
	}
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	playlistPath, err := r.hls.EnsurePlaylist(req.Context(), mediaFileID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	playlistBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write(rewriteHLSPlaylist(req, mediaFileID, playlistBytes))
}

func (r *Router) handleGetInventoryHLSPlaylist(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	playlistPath, err := r.hls.EnsureInventoryPlaylist(req.Context(), fileID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	playlistBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write(rewriteInventoryHLSPlaylist(req, fileID, playlistBytes))
}

func (r *Router) handleGetHLSArtifact(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/inventory-files/{id}/hls/{name}") {
		return
	}
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	name := req.PathValue("name")
	if _, err := r.hls.EnsurePlaylist(req.Context(), mediaFileID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	artifactPath, err := r.hls.ArtifactPath(mediaFileID, name)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if strings.HasSuffix(strings.ToLower(name), ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
	}
	http.ServeFile(w, req, artifactPath)
}

func (r *Router) handleGetInventoryHLSArtifact(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	name := req.PathValue("name")
	if _, err := r.hls.EnsureInventoryPlaylist(req.Context(), fileID); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	artifactPath, err := r.hls.InventoryArtifactPath(fileID, name)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if strings.HasSuffix(strings.ToLower(name), ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
	}
	http.ServeFile(w, req, artifactPath)
}

func (r *Router) handleStreamMediaFile(w http.ResponseWriter, req *http.Request) {
	if r.rejectLegacyMediaEndpoint(req, w, "/api/v1/inventory-files/{id}/stream") {
		return
	}
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	mediaFileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var file database.MediaFile
	if err := r.db.WithContext(req.Context()).Where("id = ? AND deleted_at IS NULL", mediaFileID).First(&file).Error; err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	provider, err := r.providerForLibrary(req.Context(), file.LibraryID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	object, err := provider.Get(req.Context(), storage.GetRequest{Path: file.StoragePath})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if object.IsDir {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("selected media file is a directory"))
		return
	}

	if provider.Name() == "local" {
		localPath := strings.TrimSpace(object.RawURL)
		if localPath == "" {
			writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("local file path unavailable for %s", file.StoragePath))
			return
		}
		if _, err := os.Stat(localPath); err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}

		if object.Modified != nil {
			w.Header().Set("Last-Modified", object.Modified.UTC().Format(http.TimeFormat))
		}
		http.ServeFile(w, req, localPath)
		return
	}

	streamURL := strings.TrimSpace(object.RawURL)
	if streamURL == "" {
		link, err := provider.Link(req.Context(), storage.LinkRequest{Path: file.StoragePath})
		if err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		streamURL = strings.TrimSpace(link.URL)
	}
	if streamURL == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("stream url unavailable for %s", file.StoragePath))
		return
	}

	upstreamReq, err := http.NewRequestWithContext(req.Context(), http.MethodGet, streamURL, nil)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	copyRequestHeader(req.Header, upstreamReq.Header, "Range")
	copyRequestHeader(req.Header, upstreamReq.Header, "If-Range")
	copyRequestHeader(req.Header, upstreamReq.Header, "If-Modified-Since")

	upstreamResp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadGateway, err)
		return
	}
	defer upstreamResp.Body.Close()

	copySelectedHeaders(upstreamResp.Header, w.Header(), proxiedStreamHeaders)
	w.WriteHeader(upstreamResp.StatusCode)
	if _, err := io.Copy(w, upstreamResp.Body); err != nil {
		log.Printf("http stream proxy path=%s error=%v", file.StoragePath, err)
	}
}

func (r *Router) handleStreamInventoryFile(w http.ResponseWriter, req *http.Request) {
	if _, err := r.requirePlaybackUser(req); err != nil {
		writeError(req.Context(), w, http.StatusUnauthorized, err)
		return
	}
	fileID, err := parseUintPathValue(req, "id")
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	var file database.InventoryFile
	if err := r.db.WithContext(req.Context()).Where("id = ? AND deleted_at IS NULL", fileID).First(&file).Error; err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	provider, err := r.providerForInventoryFile(req.Context(), file.ID)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	object, err := provider.Get(req.Context(), storage.GetRequest{Path: file.StoragePath})
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	if object.IsDir {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("selected inventory file is a directory"))
		return
	}
	if provider.Name() == "local" {
		localPath := strings.TrimSpace(object.RawURL)
		if localPath == "" {
			writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("local file path unavailable for %s", file.StoragePath))
			return
		}
		if _, err := os.Stat(localPath); err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		if object.Modified != nil {
			w.Header().Set("Last-Modified", object.Modified.UTC().Format(http.TimeFormat))
		}
		http.ServeFile(w, req, localPath)
		return
	}
	streamURL := strings.TrimSpace(object.RawURL)
	if streamURL == "" {
		link, err := provider.Link(req.Context(), storage.LinkRequest{Path: file.StoragePath})
		if err != nil {
			writeError(req.Context(), w, http.StatusBadRequest, err)
			return
		}
		streamURL = strings.TrimSpace(link.URL)
	}
	if streamURL == "" {
		writeError(req.Context(), w, http.StatusBadRequest, fmt.Errorf("stream url unavailable for %s", file.StoragePath))
		return
	}
	upstreamReq, err := http.NewRequestWithContext(req.Context(), http.MethodGet, streamURL, nil)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}
	copyRequestHeader(req.Header, upstreamReq.Header, "Range")
	copyRequestHeader(req.Header, upstreamReq.Header, "If-Range")
	copyRequestHeader(req.Header, upstreamReq.Header, "If-Modified-Since")
	upstreamResp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		writeError(req.Context(), w, http.StatusBadGateway, err)
		return
	}
	defer upstreamResp.Body.Close()
	copySelectedHeaders(upstreamResp.Header, w.Header(), proxiedStreamHeaders)
	w.WriteHeader(upstreamResp.StatusCode)
	if _, err := io.Copy(w, upstreamResp.Body); err != nil {
		log.Printf("http stream proxy inventory_path=%s error=%v", file.StoragePath, err)
	}
}

func rewriteHLSPlaylist(req *http.Request, mediaFileID uint, playlist []byte) []byte {
	scanner := bufio.NewScanner(strings.NewReader(string(playlist)))
	lines := make([]string, 0, 128)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "://") {
			lines = append(lines, line)
			continue
		}
		artifactPath := fmt.Sprintf("/api/v1/media-files/%d/hls/%s", mediaFileID, path.Base(trimmed))
		lines = append(lines, buildPlaybackURL(req, artifactPath))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func rewriteInventoryHLSPlaylist(req *http.Request, fileID uint, playlist []byte) []byte {
	scanner := bufio.NewScanner(strings.NewReader(string(playlist)))
	lines := make([]string, 0, 128)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "://") {
			lines = append(lines, line)
			continue
		}
		artifactPath := fmt.Sprintf("/api/v1/inventory-files/%d/hls/%s", fileID, path.Base(trimmed))
		lines = append(lines, buildPlaybackURL(req, artifactPath))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func buildPlaybackURL(req *http.Request, rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return trimmed
	}
	if !strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	trimmed = requestBaseURL(req) + trimmed
	token := strings.TrimSpace(req.URL.Query().Get("access_token"))
	if token == "" {
		token, _ = bearerToken(req)
	}
	if token == "" {
		return trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	query := parsed.Query()
	query.Set("access_token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func requestBaseURL(req *http.Request) string {
	scheme := "http"
	if forwardedProto := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	} else if req.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + req.Host
}

func (r *Router) providerForInventoryFile(ctx context.Context, fileID uint) (storage.Provider, error) {
	var file database.InventoryFile
	if err := r.db.WithContext(ctx).First(&file, fileID).Error; err != nil {
		return nil, err
	}
	var libraryRecord database.Library
	if err := r.db.WithContext(ctx).First(&libraryRecord, file.LibraryID).Error; err != nil {
		return nil, err
	}
	var source database.MediaSource
	if err := r.db.WithContext(ctx).First(&source, libraryRecord.MediaSourceID).Error; err != nil {
		return nil, err
	}
	return r.storage.BuildForSource(source)
}

func (r *Router) providerForLibrary(ctx context.Context, libraryID uint) (storage.Provider, error) {
	var libraryRecord database.Library
	if err := r.db.WithContext(ctx).First(&libraryRecord, libraryID).Error; err != nil {
		return nil, err
	}
	var source database.MediaSource
	if err := r.db.WithContext(ctx).First(&source, libraryRecord.MediaSourceID).Error; err != nil {
		return nil, err
	}
	return r.storage.BuildForSource(source)
}
