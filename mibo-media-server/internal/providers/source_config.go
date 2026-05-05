package providers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
)

type SourceConfig struct {
	OpenList *OpenListSourceConfig `json:"openlist,omitempty"`
}

type OpenListSourceConfig struct {
	BaseURL      string `json:"base_url"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	Token        string `json:"token,omitempty"`
	Timeout      string `json:"timeout,omitempty"`
	InsecureSkip bool   `json:"insecure_skip"`
}

type SourceConfigView struct {
	OpenList *OpenListSourceConfigView `json:"openlist,omitempty"`
}

type OpenListSourceConfigView struct {
	BaseURL      string `json:"base_url"`
	Username     string `json:"username,omitempty"`
	Timeout      string `json:"timeout,omitempty"`
	InsecureSkip bool   `json:"insecure_skip"`
	HasPassword  bool   `json:"has_password"`
	HasToken     bool   `json:"has_token"`
}

func ParseSourceConfig(raw string) (SourceConfig, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return SourceConfig{}, nil
	}

	var cfg SourceConfig
	if err := json.Unmarshal([]byte(trimmed), &cfg); err != nil {
		return SourceConfig{}, err
	}
	return cfg, nil
}

func MarshalSourceConfig(cfg SourceConfig) (string, error) {
	if cfg.OpenList == nil {
		return "", nil
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func NormalizeSourceConfig(providerName string, input *SourceConfig, fallback config.Config) (SourceConfig, error) {
	switch normalizeProviderName(providerName) {
	case "local":
		return SourceConfig{}, nil
	case "openlist":
		return normalizeOpenListSourceConfig(input, fallback)
	default:
		return SourceConfig{}, fmt.Errorf("unsupported storage provider %q", providerName)
	}
}

func normalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeOpenListSourceConfig(input *SourceConfig, fallback config.Config) (SourceConfig, error) {
	var raw OpenListSourceConfig
	if input != nil && input.OpenList != nil {
		raw = *input.OpenList
	}

	baseURL := strings.TrimRight(strings.TrimSpace(raw.BaseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(fallback.OpenList.BaseURL), "/")
	}
	username := strings.TrimSpace(raw.Username)
	if username == "" {
		username = strings.TrimSpace(fallback.OpenList.Username)
	}
	password := raw.Password
	if strings.TrimSpace(password) == "" {
		password = fallback.OpenList.Password
	}
	token := strings.TrimSpace(raw.Token)
	if token == "" {
		token = strings.TrimSpace(fallback.OpenList.Token)
	}
	timeout := strings.TrimSpace(raw.Timeout)
	if timeout == "" && fallback.OpenList.Timeout > 0 {
		timeout = fallback.OpenList.Timeout.String()
	}
	if timeout == "" {
		timeout = (60 * time.Second).String()
	}
	if _, err := time.ParseDuration(timeout); err != nil {
		return SourceConfig{}, fmt.Errorf("invalid openlist timeout %q", timeout)
	}
	insecureSkip := raw.InsecureSkip
	if !raw.InsecureSkip {
		insecureSkip = fallback.OpenList.InsecureSkip
	}

	if baseURL == "" {
		return SourceConfig{}, fmt.Errorf("openlist base_url is required")
	}
	return SourceConfig{OpenList: &OpenListSourceConfig{
		BaseURL:      baseURL,
		Username:     username,
		Password:     password,
		Token:        token,
		Timeout:      timeout,
		InsecureSkip: insecureSkip,
	}}, nil
}

func (cfg SourceConfig) Sanitized() SourceConfigView {
	view := SourceConfigView{}
	if cfg.OpenList != nil {
		view.OpenList = &OpenListSourceConfigView{
			BaseURL:      cfg.OpenList.BaseURL,
			Username:     cfg.OpenList.Username,
			Timeout:      cfg.OpenList.Timeout,
			InsecureSkip: cfg.OpenList.InsecureSkip,
			HasPassword:  strings.TrimSpace(cfg.OpenList.Password) != "",
			HasToken:     strings.TrimSpace(cfg.OpenList.Token) != "",
		}
	}
	return view
}

func (cfg SourceConfig) OpenListConfig(rootPath string) (config.OpenListConfig, error) {
	if cfg.OpenList == nil {
		return config.OpenListConfig{}, fmt.Errorf("openlist config is required")
	}
	timeout := 60 * time.Second
	if strings.TrimSpace(cfg.OpenList.Timeout) != "" {
		parsed, err := time.ParseDuration(cfg.OpenList.Timeout)
		if err != nil {
			return config.OpenListConfig{}, err
		}
		timeout = parsed
	}
	return config.OpenListConfig{
		BaseURL:      strings.TrimRight(strings.TrimSpace(cfg.OpenList.BaseURL), "/"),
		Username:     strings.TrimSpace(cfg.OpenList.Username),
		Password:     cfg.OpenList.Password,
		Token:        strings.TrimSpace(cfg.OpenList.Token),
		RootPath:     rootPath,
		Timeout:      timeout,
		InsecureSkip: cfg.OpenList.InsecureSkip,
	}, nil
}
