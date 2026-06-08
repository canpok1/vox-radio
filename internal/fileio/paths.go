package fileio

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	FileArticles  = "01_articles.json"
	FileSummaries = "02_summaries.json"
	FileRundown   = "02_rundown.json"
	FileLines     = "03_lines.json"
	FileProofread = "04_proofread.json"
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
func ProofreadPath(outDir string) string { return intermediatePath(outDir, FileProofread) }
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

// ReadJSON reads a JSON file at path and unmarshals it into v.
func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return nil
}

// DecodeYAML opens the YAML file at path and decodes it into dest.
// When strict is true, unknown fields cause an error.
func DecodeYAML(path string, dest any, strict bool) error {
	f, err := os.Open(path)
	if err != nil {
		// os.Open already includes "open <path>: <err>"; no additional wrapping needed.
		return err
	}
	defer func() { _ = f.Close() }()
	dec := yaml.NewDecoder(f)
	if strict {
		dec.KnownFields(true)
	}
	if err := dec.Decode(dest); err != nil {
		var typeErr *yaml.TypeError
		if errors.As(err, &typeErr) {
			return fmt.Errorf("decode %s: %w", path, stripGoTypeNames(typeErr))
		}
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// stripGoTypeNames removes the " in type pkg.TypeName" suffix from yaml.TypeError
// error messages so that internal Go type names are not exposed to users.
func stripGoTypeNames(e *yaml.TypeError) *yaml.TypeError {
	msgs := make([]string, len(e.Errors))
	for i, msg := range e.Errors {
		if idx := strings.Index(msg, " in type "); idx >= 0 {
			msgs[i] = msg[:idx]
		} else {
			msgs[i] = msg
		}
	}
	return &yaml.TypeError{Errors: msgs}
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
