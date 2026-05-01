package httpapi

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/playback"
	"github.com/atlan/mibo-media-server/internal/storage"
)

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
