package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/atlan/mibo-media-server/internal/config"
)

type providerRequestFailure interface {
	error
	StatusCode() int
}

type metatubeRequestFailure struct {
	statusCode int
	message    string
}

func (e metatubeRequestFailure) Error() string {
	return e.message
}

func (e metatubeRequestFailure) StatusCode() int {
	return e.statusCode
}

type metatubeEnvelope struct {
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
}

type metatubeSearchResult struct {
	ID            string          `json:"id"`
	Provider      string          `json:"provider"`
	Title         string          `json:"title"`
	OriginalTitle string          `json:"original_title"`
	Number        string          `json:"number"`
	Overview      string          `json:"overview"`
	Summary       string          `json:"summary"`
	ReleaseDate   string          `json:"release_date"`
	Date          string          `json:"date"`
	Year          int             `json:"year"`
	CoverURL      string          `json:"cover_url"`
	PosterURL     string          `json:"poster_url"`
	ThumbURL      string          `json:"thumb_url"`
	BackdropURL   string          `json:"backdrop_url"`
	Raw           json.RawMessage `json:"-"`
}

type metatubeDetailResponse struct {
	ID            string           `json:"id"`
	Provider      string           `json:"provider"`
	Title         string           `json:"title"`
	OriginalTitle string           `json:"original_title"`
	Number        string           `json:"number"`
	Overview      string           `json:"overview"`
	Summary       string           `json:"summary"`
	Description   string           `json:"description"`
	ReleaseDate   string           `json:"release_date"`
	Date          string           `json:"date"`
	Year          int              `json:"year"`
	Runtime       *int             `json:"runtime"`
	Duration      *int             `json:"duration"`
	Genres        []string         `json:"genres"`
	Tags          []string         `json:"tags"`
	Director      string           `json:"director"`
	Directors     []string         `json:"directors"`
	Actors        []metatubePerson `json:"actors"`
	Cast          []metatubePerson `json:"cast"`
	CoverURL      string           `json:"cover_url"`
	PosterURL     string           `json:"poster_url"`
	ThumbURL      string           `json:"thumb_url"`
	BackdropURL   string           `json:"backdrop_url"`
	Images        []string         `json:"images"`
	Fallback      any              `json:"fallback,omitempty"`
	Raw           json.RawMessage  `json:"-"`
}

type metatubePerson struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar_url"`
}

func (s *Service) searchMetaTube(ctx context.Context, cfg config.MetaTubeConfig, query string) ([]metatubeSearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	params := url.Values{}
	params.Set("q", query)
	if provider := strings.TrimSpace(cfg.UpstreamProviderFilter); provider != "" {
		params.Set("provider", provider)
	}
	params.Set("fallback", strconv.FormatBool(cfg.FallbackEnabled))
	var results []metatubeSearchResult
	if err := metatubeRequest(ctx, cfg, "v1/movies/search", params, &results); err != nil {
		return nil, err
	}
	for idx := range results {
		if strings.TrimSpace(results[idx].Provider) == "" {
			results[idx].Provider = strings.TrimSpace(cfg.UpstreamProviderFilter)
		}
	}
	return results, nil
}

func (s *Service) fetchMetaTubeDetail(ctx context.Context, cfg config.MetaTubeConfig, upstreamProvider string, upstreamID string) (metatubeDetailResponse, error) {
	provider := strings.TrimSpace(upstreamProvider)
	id := strings.TrimSpace(upstreamID)
	if provider == "" || id == "" {
		return metatubeDetailResponse{}, fmt.Errorf("metatube provider and id are required")
	}
	var detail metatubeDetailResponse
	if err := metatubeRequest(ctx, cfg, path.Join("v1/movies", provider, id), nil, &detail); err != nil {
		return metatubeDetailResponse{}, err
	}
	if strings.TrimSpace(detail.Provider) == "" {
		detail.Provider = provider
	}
	if strings.TrimSpace(detail.ID) == "" {
		detail.ID = id
	}
	return detail, nil
}

func metatubeRequest(ctx context.Context, cfg config.MetaTubeConfig, endpoint string, params url.Values, out any) error {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return fmt.Errorf("metatube base url is required")
	}
	requestURL := baseURL + "/" + strings.TrimLeft(endpoint, "/")
	if encoded := params.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	if token := strings.TrimSpace(cfg.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: cfg.Timeout}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return metatubeRequestError(resp.StatusCode, body)
	}
	return decodeMetaTubeEnvelope(body, out)
}

func decodeMetaTubeEnvelope(body []byte, out any) error {
	var envelope metatubeEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Data) > 0 {
		return json.Unmarshal(envelope.Data, out)
	}
	return json.Unmarshal(body, out)
}

func metatubeRequestError(statusCode int, body []byte) error {
	message := ""
	var envelope metatubeEnvelope
	if len(body) > 0 {
		_ = json.Unmarshal(body, &envelope)
		message = firstNonEmpty(envelope.Message, envelope.Error)
	}
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		if message == "" {
			message = "request was not authorized"
		}
		return metatubeRequestFailure{statusCode: statusCode, message: fmt.Sprintf("MetaTube authentication failed: %s", message)}
	case http.StatusTooManyRequests:
		if message == "" {
			message = "rate limit exceeded"
		}
		return metatubeRequestFailure{statusCode: statusCode, message: fmt.Sprintf("MetaTube rate limited: %s", message)}
	case http.StatusNotFound:
		if message == "" {
			message = "item was not found"
		}
		return metatubeRequestFailure{statusCode: statusCode, message: fmt.Sprintf("MetaTube item not found: %s", message)}
	default:
		if message != "" {
			return metatubeRequestFailure{statusCode: statusCode, message: fmt.Sprintf("MetaTube request failed(%d): %s", statusCode, message)}
		}
		return metatubeRequestFailure{statusCode: statusCode, message: fmt.Sprintf("MetaTube request failed(%d)", statusCode)}
	}
}
