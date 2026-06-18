package audio

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed *
var embedded embed.FS

// Store loads .ogg voice files bundled at build time and/or from an external directory.
type Store struct {
	externalDir string
}

func NewStore(externalDir string) *Store {
	return &Store{externalDir: strings.TrimSpace(externalDir)}
}

func (s *Store) Load(ref string) ([]byte, error) {
	path := normalizeRef(ref)
	if path == "" {
		return nil, fs.ErrNotExist
	}
	if s.externalDir != "" {
		full := filepath.Join(s.externalDir, filepath.FromSlash(path))
		if data, err := os.ReadFile(full); err == nil {
			return data, nil
		}
	}
	return fs.ReadFile(embedded, path)
}

func normalizeRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "audio/")
	ref = strings.ReplaceAll(ref, "\\", "/")
	if ref == "" || ref == "." {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(ref), ".ogg") {
		ref += ".ogg"
	}
	return filepath.ToSlash(filepath.Clean(ref))
}

// RefFromExercise returns an audio file reference when "audio" is a non-empty path.
func RefFromExercise(data map[string]interface{}) string {
	v, ok := data["audio"]
	if !ok || v == nil {
		return ""
	}
	ref := strings.TrimSpace(fmt.Sprintf("%v", v))
	if ref == "" || strings.EqualFold(ref, "null") || ref == "<nil>" {
		return ""
	}
	return ref
}
