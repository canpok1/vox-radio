package fileio_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canpok1/vox-radio/internal/fileio"
)

func TestDecodeYAML_MissingFile(t *testing.T) {
	var v any
	err := fileio.DecodeYAML(filepath.Join(t.TempDir(), "nonexistent.yaml"), &v, false)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestDecodeYAML(t *testing.T) {
	type nameOnly struct {
		Name string `yaml:"name"`
	}
	type nameAge struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}
	tests := []struct {
		name        string
		content     string
		checkResult func(t *testing.T, path string)
	}{
		{
			name:    "decodes valid YAML",
			content: "name: ずんだもん\nage: 3\n",
			checkResult: func(t *testing.T, path string) {
				var got nameAge
				if err := fileio.DecodeYAML(path, &got, false); err != nil {
					t.Fatalf("DecodeYAML: %v", err)
				}
				if got.Name != "ずんだもん" {
					t.Errorf("Name: got %q, want %q", got.Name, "ずんだもん")
				}
				if got.Age != 3 {
					t.Errorf("Age: got %d, want %d", got.Age, 3)
				}
			},
		},
		{
			name:    "unknown field ignored when non-strict",
			content: "name: foo\nunknown_key: bar\n",
			checkResult: func(t *testing.T, path string) {
				var got nameOnly
				if err := fileio.DecodeYAML(path, &got, false); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			},
		},
		{
			name:    "unknown field rejected when strict",
			content: "name: foo\nunknown_key: bar\n",
			checkResult: func(t *testing.T, path string) {
				var got nameOnly
				if err := fileio.DecodeYAML(path, &got, true); err == nil {
					t.Error("expected error for unknown field, got nil")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			tt.checkResult(t, path)
		})
	}
}

func TestClipFileName(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "clip_000.wav"},
		{1, "clip_001.wav"},
		{42, "clip_042.wav"},
		{999, "clip_999.wav"},
	}
	for _, tt := range tests {
		got := fileio.ClipFileName(tt.n)
		if got != tt.want {
			t.Errorf("ClipFileName(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestPaths(t *testing.T) {
	outDir := "/output"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"IntermediateDir", fileio.IntermediateDir(outDir), filepath.Join(outDir, "intermediate")},
		{"ClipsDir", fileio.ClipsDir(outDir), filepath.Join(outDir, "intermediate", "05_clips")},
		{"ArticlesPath", fileio.ArticlesPath(outDir), filepath.Join(outDir, "intermediate", "01_articles.json")},
		{"SummariesPath", fileio.SummariesPath(outDir), filepath.Join(outDir, "intermediate", "02_summaries.json")},
		{"RundownPath", fileio.RundownPath(outDir), filepath.Join(outDir, "intermediate", "02_rundown.json")},
		{"LinesPath", fileio.LinesPath(outDir), filepath.Join(outDir, "intermediate", "03_lines.json")},
		{"ScriptPath", fileio.ScriptPath(outDir), filepath.Join(outDir, "intermediate", "04_script.json")},
		{"EpisodePath", fileio.EpisodePath(outDir), filepath.Join(outDir, "episode.mp3")},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestEnsureDir(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "a", "b", "c")

	if err := fileio.EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}

	// idempotent
	if err := fileio.EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir (idempotent) failed: %v", err)
	}
}

func TestWriteJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "sub", "out.json")

	v := map[string]string{"key": "value"}
	if err := fileio.WriteJSON(path, v); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) == "" {
		t.Error("file is empty")
	}

	// idempotent overwrite
	if err := fileio.WriteJSON(path, v); err != nil {
		t.Fatalf("WriteJSON (overwrite) failed: %v", err)
	}
}

func TestReadJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.json")

	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	want := payload{Name: "ずんだもん", Age: 0}
	if err := fileio.WriteJSON(path, want); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got payload
	if err := fileio.ReadJSON(path, &got); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got.Name != want.Name {
		t.Errorf("Name: got %q, want %q", got.Name, want.Name)
	}
}

func TestReadJSON_MissingFile(t *testing.T) {
	var v any
	err := fileio.ReadJSON(filepath.Join(t.TempDir(), "nonexistent.json"), &v)
	if err == nil {
		t.Error("ReadJSON: expected error for missing file, got nil")
	}
}

func TestEnsureOutputDirs(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "output")

	if err := fileio.EnsureOutputDirs(outDir); err != nil {
		t.Fatalf("EnsureOutputDirs failed: %v", err)
	}
	for _, dir := range []string{
		outDir,
		fileio.IntermediateDir(outDir),
		fileio.ClipsDir(outDir),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("dir %q not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", dir)
		}
	}
}
