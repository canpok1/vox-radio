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
	FileTimeline  = "06_timeline.json"

	DirIntermediate = "intermediate"
	DirClips        = "05_clips"

	// manifestSuffix is appended to the episode base name to form the manifest
	// file name, e.g. "morning-news_ep001_manifest.json".
	manifestSuffix = "_manifest.json"
)

func ClipFileName(n int) string {
	return fmt.Sprintf("clip_%03d.wav", n)
}

// EpisodeBaseName returns the base name shared by an episode's outputs,
// e.g. "morning-news_ep001".
// When episodeNumber <= 0 (single-shot mode, see program.single_shot), the
// number is invalid and the base name is the bare programID (no "_ep{NNN}"
// suffix), e.g. "vox-radio-demo".
func EpisodeBaseName(programID string, episodeNumber int) string {
	if episodeNumber <= 0 {
		return programID
	}
	return fmt.Sprintf("%s_ep%03d", programID, episodeNumber)
}

func EpisodeFileName(programID string, episodeNumber int) string {
	return EpisodeBaseName(programID, episodeNumber) + ".mp3"
}

// EpisodeLayout resolves the output paths for a single episode produced by the
// episodegen pipeline. The manifest and intermediate files are namespaced by
// EpisodeBaseName so that repeated runs do not overwrite past episodes'
// artifacts.
type EpisodeLayout struct {
	OutDir        string
	ProgramID     string
	EpisodeNumber int
}

func (l EpisodeLayout) baseName() string {
	return EpisodeBaseName(l.ProgramID, l.EpisodeNumber)
}

// Episode returns the path to the final MP3, e.g. <out>/morning-news_ep001.mp3.
func (l EpisodeLayout) Episode() string {
	return filepath.Join(l.OutDir, l.baseName()+".mp3")
}

// Manifest returns the manifest path, e.g. <out>/morning-news_ep001_manifest.json.
func (l EpisodeLayout) Manifest() string {
	return filepath.Join(l.OutDir, l.baseName()+manifestSuffix)
}

// IntermediateDir returns the per-episode intermediate directory,
// e.g. <out>/intermediate/morning-news_ep001.
func (l EpisodeLayout) IntermediateDir() string {
	return filepath.Join(l.OutDir, DirIntermediate, l.baseName())
}

// ClipsDir returns the per-episode WAV clips directory.
func (l EpisodeLayout) ClipsDir() string {
	return filepath.Join(l.IntermediateDir(), DirClips)
}

func (l EpisodeLayout) intermediatePath(file string) string {
	return filepath.Join(l.IntermediateDir(), file)
}

func (l EpisodeLayout) Articles() string  { return l.intermediatePath(FileArticles) }
func (l EpisodeLayout) Summaries() string { return l.intermediatePath(FileSummaries) }
func (l EpisodeLayout) Rundown() string   { return l.intermediatePath(FileRundown) }
func (l EpisodeLayout) Lines() string     { return l.intermediatePath(FileLines) }
func (l EpisodeLayout) Proofread() string { return l.intermediatePath(FileProofread) }
func (l EpisodeLayout) Script() string    { return l.intermediatePath(FileScript) }
func (l EpisodeLayout) Timeline() string  { return l.intermediatePath(FileTimeline) }

// EnsureDirs creates the episode's intermediate and clips directories.
func (l EpisodeLayout) EnsureDirs() error {
	return os.MkdirAll(l.ClipsDir(), 0o755)
}

// RemoveIntermediateDir removes the episode's intermediate directory and all
// its contents. It is used when overwriting an existing episode with --force so
// that stale artifacts from a previous run do not linger.
func (l EpisodeLayout) RemoveIntermediateDir() error {
	return os.RemoveAll(l.IntermediateDir())
}

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
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
		msgs[i], _, _ = strings.Cut(msg, " in type ")
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
