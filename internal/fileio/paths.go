package fileio

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	FileArticles  = "articles.json"
	FileSummaries = "summaries.json"
	FileRundown   = "rundown.json"
	FileLines     = "lines.json"
	FileScript    = "script.json"
	FileEpisode   = "episode.mp3"

	DirIntermediate = "intermediate"
	DirClips        = "clips"
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

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func EnsureOutputDirs(outDir string) error {
	return os.MkdirAll(ClipsDir(outDir), 0o755)
}
