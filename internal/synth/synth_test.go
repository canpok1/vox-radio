package synth

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

type mockVoicevoxClient struct {
	audioQueryFn func(ctx context.Context, text string, speaker int) (*AudioQuery, error)
	synthesisFn  func(ctx context.Context, query *AudioQuery, speaker int) ([]byte, error)
}

func (m *mockVoicevoxClient) AudioQuery(ctx context.Context, text string, speaker int) (*AudioQuery, error) {
	return m.audioQueryFn(ctx, text, speaker)
}

func (m *mockVoicevoxClient) Synthesis(ctx context.Context, query *AudioQuery, speaker int) ([]byte, error) {
	return m.synthesisFn(ctx, query, speaker)
}

var fakeWAV = []byte("FAKEWAV")

func newTestSynth() *Synth {
	return &Synth{
		Client: &mockVoicevoxClient{
			audioQueryFn: func(_ context.Context, _ string, _ int) (*AudioQuery, error) {
				return &AudioQuery{SpeedScale: 1.0}, nil
			},
			synthesisFn: func(_ context.Context, _ *AudioQuery, _ int) ([]byte, error) {
				return fakeWAV, nil
			},
		},
		Config: &config.Config{
			Characters: map[string]config.CharacterConfig{
				"zundamon": {DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 3}},
				"metan":    {DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 2}},
			},
		},
		getDuration: func(_ string) (float64, error) { return 1.5, nil },
		logger:      slog.Default(),
	}
}

func TestSynth_Run_SkipsSESegments(t *testing.T) {
	s := newTestSynth()
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "こんにちは"},
			{Type: model.SegmentTypeSE, SEName: "chime"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "よろしく"},
		},
	}

	dir := t.TempDir()
	meta, err := s.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(meta.Clips) != 2 {
		t.Errorf("clips count: got %d, want 2", len(meta.Clips))
	}
}

func TestSynth_Run_NamesClipsSequentially(t *testing.T) {
	s := newTestSynth()
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "セリフ1"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "セリフ2"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "セリフ3"},
		},
	}

	dir := t.TempDir()
	meta, err := s.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"clip_000.wav", "clip_001.wav", "clip_002.wav"}
	for i, clip := range meta.Clips {
		if clip.File != want[i] {
			t.Errorf("clip[%d].File: got %s, want %s", i, clip.File, want[i])
		}
		if clip.Index != i {
			t.Errorf("clip[%d].Index: got %d, want %d", i, clip.Index, i)
		}
	}
}

func TestSynth_Run_WritesWAVFiles(t *testing.T) {
	s := newTestSynth()
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"},
		},
	}

	dir := t.TempDir()
	_, err := s.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wavPath := filepath.Join(dir, "clip_000.wav")
	got, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatalf("clip_000.wav not created: %v", err)
	}
	if string(got) != string(fakeWAV) {
		t.Errorf("wav content: got %q, want %q", got, fakeWAV)
	}
}

func TestSynth_Run_WritesClipsJSON(t *testing.T) {
	s := newTestSynth()
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"},
		},
	}

	dir := t.TempDir()
	_, err := s.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonPath := filepath.Join(dir, "clips.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("clips.json not created: %v", err)
	}

	var meta model.ClipsMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("clips.json parse error: %v", err)
	}
	if len(meta.Clips) != 1 {
		t.Errorf("clips.json clips count: got %d, want 1", len(meta.Clips))
	}
	if meta.Clips[0].DurationSec != 1.5 {
		t.Errorf("clips.json duration: got %v, want 1.5", meta.Clips[0].DurationSec)
	}
	if meta.Clips[0].SpeakerRole != "zundamon" {
		t.Errorf("clips.json speaker_role: got %s, want zundamon", meta.Clips[0].SpeakerRole)
	}
	if meta.Clips[0].Text != "テスト" {
		t.Errorf("clips.json text: got %s, want テスト", meta.Clips[0].Text)
	}
}

func TestSynth_Run_AutoCreatesOutputDir(t *testing.T) {
	s := newTestSynth()
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "テスト"},
		},
	}

	dir := filepath.Join(t.TempDir(), "nested", "clips")
	_, err := s.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("output dir not created: %s", dir)
	}
}

func TestSynth_Run_UsesSpeakerFromCharacterCatalog(t *testing.T) {
	var gotSpeakers []int
	s := &Synth{
		Client: &mockVoicevoxClient{
			audioQueryFn: func(_ context.Context, _ string, speaker int) (*AudioQuery, error) {
				gotSpeakers = append(gotSpeakers, speaker)
				return &AudioQuery{SpeedScale: 1.0}, nil
			},
			synthesisFn: func(_ context.Context, _ *AudioQuery, _ int) ([]byte, error) {
				return fakeWAV, nil
			},
		},
		Config: &config.Config{
			Characters: map[string]config.CharacterConfig{
				"zundamon": {DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 3}},
				"metan":    {DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 2}},
			},
		},
		getDuration: func(_ string) (float64, error) { return 1.0, nil },
		logger:      slog.Default(),
	}
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "A"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "B"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "unknown", Text: "C"},
		},
	}

	dir := t.TempDir()
	if _, err := s.Run(context.Background(), script, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{3, 2, 0}
	if len(gotSpeakers) != len(want) {
		t.Fatalf("speakers count: got %d, want %d", len(gotSpeakers), len(want))
	}
	for i := range want {
		if gotSpeakers[i] != want[i] {
			t.Errorf("speaker[%d]: got %d, want %d", i, gotSpeakers[i], want[i])
		}
	}
}

func TestNew_StoresConfig(t *testing.T) {
	cfg := &config.Config{
		Voicevox: config.VoicevoxConfig{URL: "http://localhost:50021"},
		Characters: map[string]config.CharacterConfig{
			"zundamon": {Name: "ずんだもん", DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 3}},
		},
	}

	s := New("http://localhost:50021", cfg)

	if s.Config != cfg {
		t.Error("New should store the config in Synth.Config")
	}
}

func TestSynth_Run_UsesStyleFromSegment(t *testing.T) {
	var gotSpeakers []int
	s := &Synth{
		Client: &mockVoicevoxClient{
			audioQueryFn: func(_ context.Context, _ string, speaker int) (*AudioQuery, error) {
				gotSpeakers = append(gotSpeakers, speaker)
				return &AudioQuery{SpeedScale: 1.0}, nil
			},
			synthesisFn: func(_ context.Context, _ *AudioQuery, _ int) ([]byte, error) {
				return fakeWAV, nil
			},
		},
		Config: &config.Config{
			Characters: map[string]config.CharacterConfig{
				"zundamon": {DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 3, "なみだめ": 76}},
			},
		},
		getDuration: func(_ string) (float64, error) { return 1.0, nil },
		logger:      slog.Default(),
	}
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Style: "なみだめ", Text: "A"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Style: "", Text: "B"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Style: "存在しない", Text: "C"},
		},
	}

	dir := t.TempDir()
	if _, err := s.Run(context.Background(), script, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{76, 3, 3}
	if len(gotSpeakers) != len(want) {
		t.Fatalf("speakers count: got %d, want %d", len(gotSpeakers), len(want))
	}
	for i := range want {
		if gotSpeakers[i] != want[i] {
			t.Errorf("speaker[%d]: got %d, want %d", i, gotSpeakers[i], want[i])
		}
	}
}

func TestSynth_Run_StoresStyleInClipMeta(t *testing.T) {
	s := newTestSynth()
	s.Config = &config.Config{
		Characters: map[string]config.CharacterConfig{
			"zundamon": {DefaultStyle: "ノーマル", Styles: map[string]int{"ノーマル": 3, "なみだめ": 76}},
		},
	}
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Style: "なみだめ", Text: "ぐすん"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Style: "", Text: "普通"},
		},
	}

	dir := t.TempDir()
	meta, err := s.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Clips[0].Style != "なみだめ" {
		t.Errorf("clip[0].Style: got %q, want なみだめ", meta.Clips[0].Style)
	}
	if meta.Clips[1].Style != "" {
		t.Errorf("clip[1].Style: got %q, want empty", meta.Clips[1].Style)
	}
}

func TestSynth_Run_EmptyClipsJSON_WhenNoSpeechSegments(t *testing.T) {
	s := newTestSynth()
	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSE, SEName: "chime"},
		},
	}

	dir := t.TempDir()
	meta, err := s.Run(context.Background(), script, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Clips == nil {
		t.Error("Clips must not be nil (want empty slice, got nil)")
	}
	if len(meta.Clips) != 0 {
		t.Errorf("clips count: got %d, want 0", len(meta.Clips))
	}

	// clips.json should have [] not null
	data, err := os.ReadFile(filepath.Join(dir, "clips.json"))
	if err != nil {
		t.Fatalf("clips.json not created: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("clips.json is not valid JSON")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("clips.json parse error: %v", err)
	}
	if string(raw["clips"]) == "null" {
		t.Error("clips.json clips field is null, want []")
	}
}

func TestSynth_Run_LogsStartAndComplete(t *testing.T) {
	s := newTestSynth()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.logger = logger

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "こんにちは"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "よろしく"},
		},
	}

	dir := t.TempDir()
	if _, err := s.Run(context.Background(), script, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "開始") {
		t.Errorf("should log start: %q", logs)
	}
	if !strings.Contains(logs, "完了") {
		t.Errorf("should log complete: %q", logs)
	}
}

func TestSynth_Run_LogsPerClipProgress(t *testing.T) {
	s := newTestSynth()

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.logger = logger

	script := model.Script{
		Segments: []model.ScriptSegment{
			{Type: model.SegmentTypeSpeech, SpeakerRole: "zundamon", Text: "セリフ1"},
			{Type: model.SegmentTypeSpeech, SpeakerRole: "metan", Text: "セリフ2"},
		},
	}

	dir := t.TempDir()
	if _, err := s.Run(context.Background(), script, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "1/2") {
		t.Errorf("should log per-clip progress (1/2): %q", logs)
	}
	if !strings.Contains(logs, "2/2") {
		t.Errorf("should log per-clip progress (2/2): %q", logs)
	}
}
