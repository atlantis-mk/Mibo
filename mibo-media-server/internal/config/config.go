package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTP     HTTPConfig
	Web      WebConfig
	Local    LocalStorageConfig
	Database DatabaseConfig
	OpenList OpenListConfig
	CORS     CORSConfig
	Metadata MetadataConfig
	FFmpeg   FFmpegConfig
	FFprobe  FFprobeConfig
	Worker   WorkerConfig
}

type HTTPConfig struct {
	Addr            string
	ShutdownTimeout time.Duration
}

type WebConfig struct {
	DistDir string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type LocalStorageConfig struct {
	RootPath string
}

type OpenListConfig struct {
	BaseURL      string
	Username     string
	Password     string
	Token        string
	RootPath     string
	Timeout      time.Duration
	InsecureSkip bool
}

type MetadataConfig struct {
	TMDB     TMDBConfig
	TVDB     TVDBConfig
	MetaTube MetaTubeConfig
}

type TMDBConfig struct {
	APIKey       string
	BaseURL      string
	ImageBaseURL string
	Language     string
	Timeout      time.Duration
}

type TVDBConfig struct {
	APIKey   string
	BaseURL  string
	Language string
	Timeout  time.Duration
}

type MetaTubeConfig struct {
	Token                  string
	BaseURL                string
	UpstreamProviderFilter string
	FallbackEnabled        bool
	Timeout                time.Duration
}

type FFmpegConfig struct {
	Enabled         bool
	Path            string
	Timeout         time.Duration
	ArtworkRootPath string
}

type FFprobeConfig struct {
	Enabled bool
	Path    string
	Timeout time.Duration
}

type WorkerConfig struct {
	Enabled               bool
	PollInterval          time.Duration
	RefreshIntervalHours  int
	ProbeWorkers          int
	WorkflowPollInterval  time.Duration
	WorkflowLeaseDuration time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Addr:            getEnv("MIBO_HTTP_ADDR", ":8080"),
			ShutdownTimeout: getDurationEnv("MIBO_HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Web: WebConfig{
			DistDir: strings.TrimSpace(os.Getenv("MIBO_WEB_DIST_DIR")),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseCSVEnv("MIBO_CORS_ALLOWED_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			Driver: strings.ToLower(getEnv("MIBO_DATABASE_DRIVER", "sqlite")),
			DSN:    getEnv("MIBO_DATABASE_DSN", filepath.Join("data", "mibo.db")),
		},
		Local: LocalStorageConfig{
			RootPath: normalizeLocalRootPath(getEnv("MIBO_LOCAL_ROOT_PATH", string(filepath.Separator))),
		},
		OpenList: OpenListConfig{
			BaseURL:      strings.TrimRight(getEnv("MIBO_OPENLIST_BASE_URL", "http://127.0.0.1:5244"), "/"),
			Username:     strings.TrimSpace(os.Getenv("MIBO_OPENLIST_USERNAME")),
			Password:     os.Getenv("MIBO_OPENLIST_PASSWORD"),
			Token:        os.Getenv("MIBO_OPENLIST_TOKEN"),
			RootPath:     normalizeRootPath(getEnv("MIBO_OPENLIST_ROOT_PATH", "/")),
			Timeout:      getDurationEnv("MIBO_OPENLIST_TIMEOUT", 60*time.Second),
			InsecureSkip: getBoolEnv("MIBO_OPENLIST_INSECURE_SKIP_VERIFY", false),
		},
		Metadata: MetadataConfig{
			TMDB: TMDBConfig{
				APIKey:       strings.TrimSpace(os.Getenv("MIBO_TMDB_API_KEY")),
				BaseURL:      strings.TrimRight(getEnv("MIBO_TMDB_BASE_URL", "https://api.themoviedb.org/3"), "/"),
				ImageBaseURL: strings.TrimRight(getEnv("MIBO_TMDB_IMAGE_BASE_URL", "https://image.tmdb.org/t/p/original"), "/"),
				Language:     getEnv("MIBO_TMDB_LANGUAGE", "zh-CN"),
				Timeout:      getDurationEnv("MIBO_TMDB_TIMEOUT", 10*time.Second),
			},
			TVDB: TVDBConfig{
				APIKey:   strings.TrimSpace(os.Getenv("MIBO_TVDB_API_KEY")),
				BaseURL:  strings.TrimRight(getEnv("MIBO_TVDB_BASE_URL", "https://api4.thetvdb.com/v4"), "/"),
				Language: getEnv("MIBO_TVDB_LANGUAGE", "en"),
				Timeout:  getDurationEnv("MIBO_TVDB_TIMEOUT", 10*time.Second),
			},
			MetaTube: MetaTubeConfig{
				Token:                  strings.TrimSpace(os.Getenv("MIBO_METATUBE_TOKEN")),
				BaseURL:                strings.TrimRight(getEnv("MIBO_METATUBE_BASE_URL", "http://127.0.0.1:8081"), "/"),
				UpstreamProviderFilter: strings.TrimSpace(os.Getenv("MIBO_METATUBE_UPSTREAM_PROVIDER_FILTER")),
				FallbackEnabled:        getBoolEnv("MIBO_METATUBE_FALLBACK_ENABLED", true),
				Timeout:                getDurationEnv("MIBO_METATUBE_TIMEOUT", 10*time.Second),
			},
		},
		FFmpeg: FFmpegConfig{
			Enabled:         getBoolEnv("MIBO_FFMPEG_ENABLED", true),
			Path:            getEnv("MIBO_FFMPEG_PATH", "ffmpeg"),
			Timeout:         getDurationEnv("MIBO_FFMPEG_TIMEOUT", 2*time.Minute),
			ArtworkRootPath: normalizeWorkPath(getEnv("MIBO_ARTWORK_ROOT_PATH", filepath.Join("tmp", "artwork"))),
		},
		FFprobe: FFprobeConfig{
			Enabled: getBoolEnv("MIBO_FFPROBE_ENABLED", true),
			Path:    getEnv("MIBO_FFPROBE_PATH", "ffprobe"),
			Timeout: getDurationEnv("MIBO_FFPROBE_TIMEOUT", 30*time.Second),
		},
		Worker: WorkerConfig{
			Enabled:               getBoolEnv("MIBO_WORKER_ENABLED", true),
			PollInterval:          getDurationEnv("MIBO_WORKER_POLL_INTERVAL", 2*time.Second),
			RefreshIntervalHours:  getIntEnv("MIBO_WORKER_REFRESH_INTERVAL_HOURS", 0),
			ProbeWorkers:          getBoundedIntEnv("MIBO_PROBE_WORKERS", 2, 1, 8),
			WorkflowPollInterval:  getDurationEnv("MIBO_WORKFLOW_POLL_INTERVAL", 2*time.Second),
			WorkflowLeaseDuration: getDurationEnv("MIBO_WORKFLOW_LEASE_DURATION", time.Minute),
		},
	}

	if cfg.Database.Driver != "sqlite" && cfg.Database.Driver != "postgres" {
		return Config{}, fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getBoundedIntEnv(key string, fallback int, minValue int, maxValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < minValue {
		return minValue
	}
	if maxValue > 0 && parsed > maxValue {
		return maxValue
	}
	return parsed
}

func parseCSVEnv(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func normalizeRootPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

func normalizeLocalRootPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return string(filepath.Separator)
	}
	if !filepath.IsAbs(trimmed) {
		if absolute, err := filepath.Abs(trimmed); err == nil {
			trimmed = absolute
		}
	}
	return filepath.Clean(trimmed)
}

func normalizeWorkPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return filepath.Clean("tmp")
	}
	return filepath.Clean(trimmed)
}
