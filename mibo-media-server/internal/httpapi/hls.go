package httpapi

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atlan/mibo-media-server/internal/config"
	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/providers"
	"github.com/atlan/mibo-media-server/internal/storage"
	"gorm.io/gorm"
)

type hlsService struct {
	cfg     config.Config
	db      *gorm.DB
	storage *providers.Registry
	locks   sync.Map
}

func newHLSService(cfg config.Config, db *gorm.DB, registry *providers.Registry) *hlsService {
	return &hlsService{cfg: cfg, db: db, storage: registry}
}

func (s *hlsService) Enabled() bool {
	return s.cfg.FFmpeg.Enabled && s.cfg.HLS.Enabled
}

func (s *hlsService) EnsureInventoryPlaylist(ctx context.Context, fileID uint) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("hls playback is disabled")
	}

	s.cleanupExpiredArtifacts()

	lock := s.fileLock(inventoryArtifactKey(fileID))
	lock.Lock()
	defer lock.Unlock()

	artifactDir := s.inventoryArtifactDir(fileID)
	playlistPath := filepath.Join(artifactDir, "index.m3u8")
	if fileExists(playlistPath) {
		return playlistPath, nil
	}
	if err := os.MkdirAll(s.cfg.HLS.RootPath, 0o755); err != nil {
		return "", err
	}

	inputSource, err := s.resolveInventoryInputSource(ctx, fileID)
	if err != nil {
		return "", err
	}

	tempDir := artifactDir + ".tmp"
	if err := os.RemoveAll(tempDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", err
	}

	if err := s.runFFmpeg(ctx, inputSource, tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}
	if !fileExists(filepath.Join(tempDir, "index.m3u8")) {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("ffmpeg did not produce hls playlist")
	}

	if err := os.RemoveAll(artifactDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}
	if err := os.Rename(tempDir, artifactDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}

	return playlistPath, nil
}

func (s *hlsService) InventoryArtifactPath(fileID uint, name string) (string, error) {
	cleanName := filepath.Base(strings.TrimSpace(name))
	if cleanName == "." || cleanName == "" || cleanName != strings.TrimSpace(name) {
		return "", fmt.Errorf("invalid hls artifact name")
	}
	artifactPath := filepath.Join(s.inventoryArtifactDir(fileID), cleanName)
	if !isWithinRoot(s.inventoryArtifactDir(fileID), artifactPath) {
		return "", fmt.Errorf("invalid hls artifact path")
	}
	return artifactPath, nil
}

func (s *hlsService) runFFmpeg(ctx context.Context, inputSource, outputDir string) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.FFmpeg.Timeout)
	defer cancel()

	playlistPath := filepath.Join(outputDir, "index.m3u8")
	segmentPattern := filepath.Join(outputDir, "segment_%03d.ts")
	copyArgs := []string{
		"-y",
		"-nostdin",
		"-i", inputSource,
		"-map", "0:v:0?",
		"-map", "0:a?",
		"-c", "copy",
		"-sn",
		"-f", "hls",
		"-hls_time", strconv.Itoa(s.cfg.HLS.SegmentDuration),
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	}
	if err := s.runFFmpegCommand(ctx, copyArgs); err == nil {
		return nil
	}

	transcodeArgs := []string{
		"-y",
		"-nostdin",
		"-i", inputSource,
		"-map", "0:v:0?",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "192k",
		"-ac", "2",
		"-sn",
		"-f", "hls",
		"-hls_time", strconv.Itoa(s.cfg.HLS.SegmentDuration),
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	}
	return s.runFFmpegCommand(ctx, transcodeArgs)
}

func (s *hlsService) runFFmpegCommand(ctx context.Context, args []string) error {
	command := exec.CommandContext(ctx, s.cfg.FFmpeg.Path, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg hls transcode failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *hlsService) resolveInventoryInputSource(ctx context.Context, fileID uint) (string, error) {
	var file database.InventoryFile
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", fileID).
		First(&file).Error; err != nil {
		return "", err
	}

	var libraryRecord database.Library
	if err := s.db.WithContext(ctx).First(&libraryRecord, file.LibraryID).Error; err != nil {
		return "", err
	}
	var source database.MediaSource
	if err := s.db.WithContext(ctx).First(&source, libraryRecord.MediaSourceID).Error; err != nil {
		return "", err
	}
	provider, err := s.storage.BuildForSource(source)
	if err != nil {
		return "", err
	}
	object, err := provider.Get(ctx, storage.GetRequest{Path: file.StoragePath})
	if err != nil {
		return "", err
	}
	if object.IsDir {
		return "", fmt.Errorf("selected inventory file is a directory")
	}
	inputSource := strings.TrimSpace(object.RawURL)
	if inputSource != "" {
		return inputSource, nil
	}
	link, err := provider.Link(ctx, storage.LinkRequest{Path: file.StoragePath})
	if err != nil {
		return "", err
	}
	inputSource = strings.TrimSpace(link.URL)
	if inputSource == "" {
		return "", fmt.Errorf("media input source unavailable for %s", file.StoragePath)
	}
	return inputSource, nil
}

func (s *hlsService) cleanupExpiredArtifacts() {
	if s.cfg.HLS.CleanupAge <= 0 {
		return
	}

	entries, err := os.ReadDir(s.cfg.HLS.RootPath)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-s.cfg.HLS.CleanupAge)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(s.cfg.HLS.RootPath, entry.Name()))
	}
}

func (s *hlsService) inventoryArtifactDir(fileID uint) string {
	return filepath.Join(s.cfg.HLS.RootPath, inventoryArtifactKey(fileID))
}

func (s *hlsService) fileLock(key string) *sync.Mutex {
	lock, _ := s.locks.LoadOrStore(key, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func inventoryArtifactKey(fileID uint) string {
	return "inventory-" + strconv.FormatUint(uint64(fileID), 10)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isWithinRoot(rootPath, targetPath string) bool {
	cleanRoot := filepath.Clean(rootPath)
	cleanTarget := filepath.Clean(targetPath)
	rel, err := filepath.Rel(cleanRoot, cleanTarget)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "..")
}
