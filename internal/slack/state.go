package slack

import (
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-radio/internal/fileio"
)

// PostState records the progress of a slackpost execution for idempotent resume.
type PostState struct {
	AudioFile     string `json:"audio_file"`
	EpisodeNumber int    `json:"episode_number"`
	Channel       string `json:"channel"`
	FileID        string `json:"file_id"`
	ThreadTS      string `json:"thread_ts"`
	Replied       bool   `json:"replied"`
}

// DefaultStatePath returns the default state file path derived from manifestPath.
// e.g. output/manifest.json → output/manifest.slackpost-state.json
func DefaultStatePath(manifestPath string) string {
	dir := filepath.Dir(manifestPath)
	base := filepath.Base(manifestPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, stem+".slackpost-state.json")
}

func loadState(path string) (*PostState, error) {
	var s PostState
	if err := fileio.ReadJSON(path, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func saveState(path string, s PostState) error {
	return fileio.WriteJSON(path, s)
}
