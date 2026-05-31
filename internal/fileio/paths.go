package fileio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	FileArticles  = "01_articles.json"
	FileSummaries = "02_summaries.json"
	FileRundown   = "02_rundown.json"
	FileLines     = "03_lines.json"
	FileScript    = "04_script.json"
	FileEpisode   = "episode.mp3"
	FileManifest  = "manifest.json"

	DirIntermediate = "intermediate"
	DirClips        = "05_clips"
)

func ClipFileName(n int) string {
	return fmt.Sprintf("clip_%03d.wav", n)
}

func IntermediateDir(outDir string) string {
	return filepath.Join(outDir, DirIntermediate)
}

func ClipsDir(outDir string) string {
	return filepath.Join(outDir, DirIntermediate, DirClips)
}

func intermediatePath(outDir, file string) string {
	return filepath.Join(outDir, DirIntermediate, file)
}

func ArticlesPath(outDir string) string  { return intermediatePath(outDir, FileArticles) }
func SummariesPath(outDir string) string { return intermediatePath(outDir, FileSummaries) }
func RundownPath(outDir string) string   { return intermediatePath(outDir, FileRundown) }
func LinesPath(outDir string) string     { return intermediatePath(outDir, FileLines) }
func ScriptPath(outDir string) string    { return intermediatePath(outDir, FileScript) }

func EpisodePath(outDir string) string {
	return filepath.Join(outDir, FileEpisode)
}

func ManifestPath(outDir string) string {
	return filepath.Join(outDir, FileManifest)
}

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func EnsureOutputDirs(outDir string) error {
	return os.MkdirAll(ClipsDir(outDir), 0o755)
}

// WriteJSON marshals v to indented JSON and writes it to path,
// creating parent directories as needed.
func WriteJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
