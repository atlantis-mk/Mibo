package openlist

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/storage"
)

type Adapter struct {
	baseURL  string
	username string
	password string
	token    string
	client   *http.Client
	tokenMu  sync.Mutex
}

type openListObject struct {
	Name         string            `json:"name"`
	IsDir        bool              `json:"is_dir"`
	Size         int64             `json:"size"`
	Created      *time.Time        `json:"created"`
	Modified     *time.Time        `json:"modified"`
	RawURL       string            `json:"raw_url"`
	ThumbnailURL string            `json:"thumb"`
	Provider     string            `json:"provider"`
	HashInfo     map[string]string `json:"hash_info"`
	ObjectType   json.RawMessage   `json:"type"`
	Sign         string            `json:"sign"`
	Related      []openListObject  `json:"related"`
	MountDetails *json.RawMessage  `json:"mount_details"`
	Readme       *json.RawMessage  `json:"readme"`
	Header       *json.RawMessage  `json:"header"`
	CanWrite     *json.RawMessage  `json:"write"`
	Upload       *json.RawMessage  `json:"upload"`
}

func New(cfg config.OpenListConfig) *Adapter {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkip}

	return &Adapter{
		baseURL:  cfg.BaseURL,
		username: strings.TrimSpace(cfg.Username),
		password: cfg.Password,
		token:    strings.TrimSpace(cfg.Token),
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
	}
}

func (a *Adapter) Name() string {
	return "openlist"
}

func (a *Adapter) List(ctx context.Context, req storage.ListRequest) ([]storage.Object, error) {
	request := map[string]any{
		"path":    normalizePath(req.Path),
		"refresh": req.Refresh,
		"page":    max(req.Page, 1),
		"per_page": func() int {
			if req.PerPage <= 0 {
				return 1000
			}
			return req.PerPage
		}(),
	}

	var response struct {
		Provider string           `json:"provider"`
		Content  []openListObject `json:"content"`
	}

	if err := a.post(ctx, "/api/fs/list", request, &response); err != nil {
		return nil, err
	}

	objects := make([]storage.Object, 0, len(response.Content))
	for _, item := range response.Content {
		objects = append(objects, mapOpenListObject(item, joinPath(req.Path, item.Name), response.Provider, ""))
	}

	return objects, nil
}

func (a *Adapter) Get(ctx context.Context, req storage.GetRequest) (storage.Object, error) {
	request := map[string]any{"path": normalizePath(req.Path)}

	var response openListObject

	if err := a.post(ctx, "/api/fs/get", request, &response); err != nil {
		return storage.Object{}, err
	}

	objectPath := normalizePath(req.Path)
	return mapOpenListObject(response, objectPath, response.Provider, relatedParentPath(objectPath)), nil
}

func (a *Adapter) Link(ctx context.Context, req storage.LinkRequest) (storage.LinkResult, error) {
	request := map[string]any{"path": normalizePath(req.Path)}

	var response struct {
		URL string `json:"url"`
	}

	if err := a.post(ctx, "/api/fs/link", request, &response); err != nil {
		return storage.LinkResult{}, err
	}

	return storage.LinkResult{URL: response.URL}, nil
}

func (a *Adapter) ResolveStorage(ctx context.Context, req storage.ResolveStorageRequest) (storage.ResolvedStorage, error) {
	object, err := a.Get(ctx, storage.GetRequest{Path: req.Path})
	if err != nil {
		return storage.ResolvedStorage{}, err
	}

	caps, err := a.Capabilities(ctx)
	if err != nil {
		return storage.ResolvedStorage{}, err
	}

	return storage.ResolvedStorage{
		Provider: a.Name(),
		Path:     normalizePath(req.Path),
		Object:   object,
		Caps:     caps,
	}, nil
}

func (a *Adapter) Capabilities(context.Context) (storage.Capabilities, error) {
	return storage.Capabilities{
		CanList: true,
		CanGet:  true,
		CanLink: true,
	}, nil
}

func (a *Adapter) post(ctx context.Context, endpoint string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	token, err := a.authorizationToken(ctx)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("openlist request failed with status %d", resp.StatusCode)
	}

	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}

	if envelope.Code != http.StatusOK {
		if envelope.Message == "" {
			return fmt.Errorf("openlist request failed with code %d", envelope.Code)
		}
		return fmt.Errorf("openlist request failed: %s", envelope.Message)
	}

	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}

	return json.Unmarshal(envelope.Data, out)
}

func (a *Adapter) authorizationToken(ctx context.Context) (string, error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.token != "" {
		return a.token, nil
	}
	if a.username == "" || strings.TrimSpace(a.password) == "" {
		return "", nil
	}

	requestBody := map[string]string{
		"username": a.username,
		"password": a.password,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}
	if envelope.Code != http.StatusOK {
		if envelope.Message == "" {
			return "", fmt.Errorf("openlist login failed with code %d", envelope.Code)
		}
		return "", fmt.Errorf("openlist login failed: %s", envelope.Message)
	}
	if strings.TrimSpace(envelope.Data.Token) == "" {
		return "", fmt.Errorf("openlist login returned empty token")
	}

	a.token = strings.TrimSpace(envelope.Data.Token)
	return a.token, nil
}

func joinPath(parent, name string) string {
	cleanParent := normalizePath(parent)
	if cleanParent == "/" {
		return normalizePath(name)
	}
	return normalizePath(path.Join(cleanParent, name))
}

func normalizePath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || trimmed == "." {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return path.Clean(trimmed)
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func cloneHashInfo(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		cloned[trimmedKey] = trimmedValue
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func mapOpenListObject(input openListObject, objectPath string, fallbackProvider string, relatedParent string) storage.Object {
	provider := strings.TrimSpace(input.Provider)
	if provider == "" {
		provider = strings.TrimSpace(fallbackProvider)
	}
	object := storage.Object{
		Name:         input.Name,
		Path:         normalizePath(objectPath),
		IsDir:        input.IsDir,
		Size:         input.Size,
		Created:      input.Created,
		Modified:     input.Modified,
		RawURL:       strings.TrimSpace(input.RawURL),
		ThumbnailURL: strings.TrimSpace(input.ThumbnailURL),
		Provider:     provider,
		HashInfo:     cloneHashInfo(input.HashInfo),
		ObjectType:   openListObjectType(input.ObjectType),
		Sign:         strings.TrimSpace(input.Sign),
		ProviderMeta: openListProviderMeta(input),
	}
	if relatedParent != "" && len(input.Related) > 0 {
		object.Related = make([]storage.Object, 0, len(input.Related))
		for _, related := range input.Related {
			relatedPath := joinPath(relatedParent, related.Name)
			object.Related = append(object.Related, mapOpenListObject(related, relatedPath, provider, ""))
		}
	}
	return object
}

func openListObjectType(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}
	return trimmed
}

func openListProviderMeta(input openListObject) map[string]string {
	meta := make(map[string]string)
	if strings.TrimSpace(input.Sign) != "" {
		meta["has_sign"] = "true"
	}
	if input.MountDetails != nil {
		meta["has_mount_details"] = "true"
	}
	if input.Readme != nil {
		meta["has_readme"] = "true"
	}
	if input.Header != nil {
		meta["has_header"] = "true"
	}
	if input.CanWrite != nil {
		meta["has_write_flag"] = "true"
	}
	if input.Upload != nil {
		meta["has_upload_metadata"] = "true"
	}
	if len(input.Related) > 0 {
		meta["has_related"] = "true"
	}
	return storage.CloneStringMap(meta)
}

func relatedParentPath(objectPath string) string {
	parent := path.Dir(normalizePath(objectPath))
	if parent == "." {
		return "/"
	}
	return normalizePath(parent)
}
