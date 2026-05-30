package ghpages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
)

// Hosting implements hosting.Hosting using a local git working tree of the gh-pages branch.
// Call Push to commit all staged files as a fresh orphan commit and force-push to origin.
type Hosting struct {
	dir     string
	baseURL string
}

// New creates a Hosting that stages files in dir and constructs public URLs with the given baseURL prefix.
// dir should be a local checkout of the gh-pages branch (or a git-initialized working tree).
func New(dir, baseURL string) *Hosting {
	return &Hosting{
		dir:     dir,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Push creates a fresh orphan commit from all files in dir and force-pushes to origin gh-pages.
func (h *Hosting) Push(ctx context.Context) error {
	type step struct {
		name string
		args []string
	}

	steps := []step{
		{"config email", []string{"config", "user.email", "github-actions[bot]@users.noreply.github.com"}},
		{"config name", []string{"config", "user.name", "github-actions[bot]"}},
		{"checkout orphan", []string{"checkout", "--orphan", "gh-pages-deploy"}},
		{"add all", []string{"add", "-A"}},
		{"commit", []string{"commit", "-m", "Deploy to gh-pages"}},
		{"push", []string{"push", "--force", "origin", "HEAD:gh-pages"}},
	}

	for _, s := range steps {
		cmd := exec.CommandContext(ctx, "git", s.args...)
		cmd.Dir = h.dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %w\n%s", s.name, err, out)
		}
	}
	return nil
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
