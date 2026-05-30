package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
)

// Hosting implements hosting.Hosting using the local filesystem.
type Hosting struct {
	dir     string
	baseURL string
}

// New creates a local Hosting that stores files under dir and constructs
// public URLs with the given baseURL prefix (trailing slash is trimmed).
func New(dir, baseURL string) *Hosting {
	return &Hosting{
		dir:     dir,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (h *Hosting) PutAudio(_ context.Context, name string, r io.Reader) (string, error) {
	audioDir := filepath.Join(h.dir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return "", fmt.Errorf("create audio dir: %w", err)
	}

	path := filepath.Join(audioDir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create audio file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("write audio: %w", err)
	}

	return h.baseURL + "/audio/" + name, nil
}

func (h *Hosting) PutFeed(_ context.Context, feedXML []byte) (string, error) {
	if err := os.MkdirAll(h.dir, 0o755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	path := filepath.Join(h.dir, "feed.xml")
	if err := os.WriteFile(path, feedXML, 0o644); err != nil {
		return "", fmt.Errorf("write feed.xml: %w", err)
	}

	return h.baseURL + "/feed.xml", nil
}

func (h *Hosting) LoadEpisodes(_ context.Context) (model.Episodes, error) {
	path := filepath.Join(h.dir, "episodes.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return model.Episodes{Episodes: make([]model.Episode, 0)}, nil
	}
	if err != nil {
		return model.Episodes{}, fmt.Errorf("read episodes.json: %w", err)
	}

	var eps model.Episodes
	if err := json.Unmarshal(data, &eps); err != nil {
		return model.Episodes{}, fmt.Errorf("parse episodes.json: %w", err)
	}
	if eps.Episodes == nil {
		eps.Episodes = make([]model.Episode, 0)
	}
	return eps, nil
}

func (h *Hosting) SaveEpisodes(_ context.Context, e model.Episodes) error {
	if err := os.MkdirAll(h.dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal episodes: %w", err)
	}

	path := filepath.Join(h.dir, "episodes.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write episodes.json: %w", err)
	}
	return nil
}

func (h *Hosting) DeleteAudio(_ context.Context, name string) error {
	path := filepath.Join(h.dir, "audio", name)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete audio %s: %w", name, err)
	}
	return nil
}
