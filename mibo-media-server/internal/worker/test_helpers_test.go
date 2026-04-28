package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFakeFFprobe(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffprobe")
	script := `#!/bin/sh
cat <<'JSON'
{"format":{"duration":"1200.0","bit_rate":"1000000"},"streams":[{"codec_type":"video","codec_name":"h264","width":1920,"height":1080},{"codec_type":"audio","codec_name":"aac","channels":2,"tags":{"language":"eng"}},{"codec_type":"subtitle","codec_name":"srt","tags":{"language":"eng"}}]}
JSON
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ffprobe: %v", err)
	}
	return path
}

func writeSlowFakeFFprobe(t *testing.T, delay time.Duration) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ffprobe")
	script := fmt.Sprintf(`#!/bin/sh
sleep %.3f
cat <<'JSON'
{"format":{"duration":"1200.0","bit_rate":"1000000"},"streams":[{"codec_type":"video","codec_name":"h264","width":1920,"height":1080},{"codec_type":"audio","codec_name":"aac","channels":2,"tags":{"language":"eng"}},{"codec_type":"subtitle","codec_name":"srt","tags":{"language":"eng"}}]}
JSON
`, delay.Seconds())
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write slow fake ffprobe: %v", err)
	}
	return path
}
