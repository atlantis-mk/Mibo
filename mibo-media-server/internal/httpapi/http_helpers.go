package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/metadata"
)

func decodeJSON(req *http.Request, out any) error {
	defer req.Body.Close()

	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}

	if decoder.More() {
		return errors.New("request body must contain a single JSON document")
	}

	return nil
}

func writeError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	writeResponse(ctx, w, status, nil, &responseError{
		Code:    codeForStatus(status),
		Message: err.Error(),
	})
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	writeResponse(ctx, w, status, data, nil)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestID := newRequestID()
		ctx := context.WithValue(req.Context(), requestIDContextKey, requestID)
		req = req.WithContext(ctx)

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		recorder.Header().Set("X-Request-ID", requestID)
		startedAt := time.Now()

		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("http request_id=%s method=%s path=%s panic=%v", requestID, req.Method, req.URL.Path, recovered)
				writeError(ctx, recorder, http.StatusInternalServerError, errors.New("internal server error"))
			}
			log.Printf(
				"http request_id=%s method=%s path=%s status=%d duration=%s",
				requestID,
				req.Method,
				req.URL.Path,
				recorder.status,
				time.Since(startedAt).Round(time.Millisecond),
			)
		}()

		next.ServeHTTP(recorder, req)
	})
}

func corsMiddleware(cfg config.CORSConfig, next http.Handler) http.Handler {
	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		origin := req.Header.Get("Origin")
		switch {
		case len(allowedOrigins) == 1 && allowedOrigins[0] == "*":
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case origin != "" && containsOrigin(allowedOrigins, origin):
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, req)
	})
}

type contextKey string

const requestIDContextKey contextKey = "request_id"

type envelope struct {
	RequestID string         `json:"request_id"`
	Data      any            `json:"data,omitempty"`
	Error     *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeResponse(ctx context.Context, w http.ResponseWriter, status int, data any, err *responseError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{
		RequestID: requestIDFromContext(ctx),
		Data:      data,
		Error:     err,
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		return "internal_error"
	}
}

func newRequestID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(raw)
}

func parseUintPathValue(req *http.Request, key string) (uint, error) {
	value := req.PathValue(key)
	if value == "" {
		return 0, fmt.Errorf("missing path parameter %q", key)
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid path parameter %q", key)
	}

	return uint(parsed), nil
}

func parseIntPathValue(req *http.Request, key string) (int, error) {
	value := req.PathValue(key)
	if value == "" {
		return 0, fmt.Errorf("missing path parameter %q", key)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid path parameter %q", key)
	}

	return parsed, nil
}

func parseOptionalUintQuery(req *http.Request, key string) (uint, error) {
	value := req.URL.Query().Get(key)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid query parameter %q", key)
	}
	return uint(parsed), nil
}

func errMissingJSONField(key string) error {
	return fmt.Errorf("missing JSON field %q", key)
}

func parseOptionalIntQuery(req *http.Request, key string) (int, bool, error) {
	value := strings.TrimSpace(req.URL.Query().Get(key))
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, fmt.Errorf("invalid query parameter %q", key)
	}
	return parsed, true, nil
}

func containsOrigin(origins []string, target string) bool {
	for _, origin := range origins {
		if origin == target {
			return true
		}
	}
	return false
}

func bearerToken(req *http.Request) (string, error) {
	authorization := strings.TrimSpace(req.Header.Get("Authorization"))
	if authorization == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return "", fmt.Errorf("authorization header must use Bearer token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
	if token == "" {
		return "", fmt.Errorf("missing bearer token")
	}
	return token, nil
}

func playbackToken(req *http.Request) (string, error) {
	token := strings.TrimSpace(req.URL.Query().Get("access_token"))
	if token != "" {
		return token, nil
	}
	return bearerToken(req)
}

func copyRequestHeader(from http.Header, to http.Header, key string) {
	value := strings.TrimSpace(from.Get(key))
	if value != "" {
		to.Set(key, value)
	}
}

func copySelectedHeaders(from http.Header, to http.Header, keys []string) {
	for _, key := range keys {
		for _, value := range from.Values(key) {
			to.Add(key, value)
		}
	}
}

func (r *Router) requireUser(req *http.Request) (database.User, error) {
	token, err := bearerToken(req)
	if err != nil {
		return database.User{}, err
	}
	return r.auth.Authenticate(req.Context(), token)
}

func (r *Router) requireAdminUser(req *http.Request) (database.User, error) {
	user, err := r.requireUser(req)
	if err != nil {
		return database.User{}, err
	}
	if strings.ToLower(strings.TrimSpace(user.Role)) != "admin" {
		return database.User{}, errors.New("admin access required")
	}
	return user, nil
}

func (r *Router) optionalUser(req *http.Request) (*database.User, error) {
	token, err := bearerToken(req)
	if err != nil {
		return nil, nil
	}
	user, err := r.auth.Authenticate(req.Context(), token)
	if err != nil {
		return nil, nil
	}
	return &user, nil
}

func (r *Router) requirePlaybackUser(req *http.Request) (database.User, error) {
	token, err := playbackToken(req)
	if err != nil {
		return database.User{}, err
	}
	return r.auth.Authenticate(req.Context(), token)
}

func validateManualSearchInput(input metadata.ManualSearchInput) error {
	if title := strings.TrimSpace(input.Title); len(title) > 512 {
		return fmt.Errorf("title must be 512 characters or fewer")
	}
	if input.Year != nil && (*input.Year < 1800 || *input.Year > 3000) {
		return fmt.Errorf("year must be between 1800 and 3000")
	}
	if len(strings.TrimSpace(input.IMDbID)) > 64 {
		return fmt.Errorf("imdb_id must be 64 characters or fewer")
	}
	if len(strings.TrimSpace(input.TMDBID)) > 64 {
		return fmt.Errorf("tmdb_id must be 64 characters or fewer")
	}
	if len(strings.TrimSpace(input.TVDBID)) > 64 {
		return fmt.Errorf("tvdb_id must be 64 characters or fewer")
	}
	return nil
}

func validateManualMetadataInput(input metadata.ManualMetadataInput) error {
	if title := strings.TrimSpace(input.Title); title == "" {
		return fmt.Errorf("title is required")
	} else if len(title) > 512 {
		return fmt.Errorf("title must be 512 characters or fewer")
	}
	if len(strings.TrimSpace(input.OriginalTitle)) > 512 {
		return fmt.Errorf("original_title must be 512 characters or fewer")
	}
	if input.Year != nil && (*input.Year < 1800 || *input.Year > 3000) {
		return fmt.Errorf("year must be between 1800 and 3000")
	}
	if len(strings.TrimSpace(input.Overview)) > 20000 {
		return fmt.Errorf("overview must be 20000 characters or fewer")
	}
	if err := validateOptionalHTTPURL("poster_url", input.PosterURL); err != nil {
		return err
	}
	if err := validateOptionalHTTPURL("backdrop_url", input.BackdropURL); err != nil {
		return err
	}
	return nil
}

func validateOptionalHTTPURL(field, raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > 2048 {
		return fmt.Errorf("%s must be 2048 characters or fewer", field)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid absolute URL", field)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	return nil
}
