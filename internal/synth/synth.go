package synth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/model"
)

// Synth synthesizes speech segments from a script using the VOICEVOX HTTP API
type Synth struct {
	Client      VoicevoxClient
	Config      *config.Config
	getDuration func(path string) (float64, error)
	logger      *slog.Logger
}

// Option configures a Synth.
type Option func(*Synth)

// WithLogger sets the logger used for progress messages.
func WithLogger(l *slog.Logger) Option {
	return func(s *Synth) { s.logger = l }
}

// New creates a new Synth with an HTTP VOICEVOX client.
func New(engineURL string, cfg *config.Config, opts ...Option) *Synth {
	s := &Synth{
		Client:      NewClient(engineURL),
		Config:      cfg,
		getDuration: mediainfo.Duration,
		logger:      slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run synthesizes all speech segments and saves clip_NNN.wav files to outDir.
// It also writes clips.json with metadata including durations.
func (s *Synth) Run(ctx context.Context, script model.Script, outDir string) (*model.ClipsMeta, error) {
	logger := s.logger.With("step", "synth")
	start := time.Now()

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	speechSegs := make([]model.ScriptSegment, 0, len(script.Segments))
	for _, seg := range script.Segments {
		if seg.Type == model.SegmentTypeSpeech {
			speechSegs = append(speechSegs, seg)
		}
	}

	logger.Info(fmt.Sprintf("開始 (%dクリップ)", len(speechSegs)))

	var presets config.VoicevoxPresets
	if s.Config != nil {
		presets = s.Config.Voicevox.EffectivePresets()
	}

	clips := make([]model.ClipMeta, 0, len(speechSegs))

	for i, seg := range speechSegs {
		logger.Info(fmt.Sprintf("クリップを合成中 (%d/%d)", i+1, len(speechSegs)))
		logger.Debug("クリップ詳細", "speaker", seg.SpeakerRole, "style", seg.Style, "text_chars", utf8.RuneCountInString(seg.Text))

		speakerID := s.resolveSpeakerID(seg.SpeakerRole, seg.Style)
		clipFile := fmt.Sprintf("clip_%03d.wav", i)
		clipPath := filepath.Join(outDir, clipFile)

		if err := s.synthesize(ctx, seg, speakerID, clipPath, presets); err != nil {
			return nil, fmt.Errorf("synthesize clip %d: %w", i, err)
		}

		dur, err := s.getDuration(clipPath)
		if err != nil {
			return nil, fmt.Errorf("get duration of %s: %w", clipFile, err)
		}

		logger.Debug("クリップ出力", "file", clipFile, "duration_sec", dur)

		clips = append(clips, model.ClipMeta{
			Index:       i,
			File:        clipFile,
			DurationSec: dur,
			SpeakerRole: seg.SpeakerRole,
			Style:       seg.Style,
			Text:        seg.Text,
			CornerID:    seg.CornerID,
		})
	}

	meta := &model.ClipsMeta{Clips: clips}

	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal clips meta: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "clips.json"), b, 0o644); err != nil {
		return nil, fmt.Errorf("write clips.json: %w", err)
	}

	logger.Info(fmt.Sprintf("完了 (%dクリップ, %.1fs)", len(clips), time.Since(start).Seconds()))

	return meta, nil
}

func (s *Synth) synthesize(ctx context.Context, seg model.ScriptSegment, speakerID int, outPath string, presets config.VoicevoxPresets) error {
	query, err := s.Client.AudioQuery(ctx, seg.Text, speakerID)
	if err != nil {
		return fmt.Errorf("audio query: %w", err)
	}

	if v, ok := presets.ResolveIntonation(seg.Intonation); ok {
		query.IntonationScale = v
	}
	if v, ok := presets.ResolvePitch(seg.Pitch); ok {
		query.PitchScale = v
	}
	if v, ok := presets.ResolveSpeed(seg.Speed); ok {
		query.SpeedScale = v
	}

	wavBytes, err := s.Client.Synthesis(ctx, query, speakerID)
	if err != nil {
		return fmt.Errorf("synthesis: %w", err)
	}

	if err := os.WriteFile(outPath, wavBytes, 0o644); err != nil {
		return fmt.Errorf("write wav: %w", err)
	}
	return nil
}

// resolveSpeakerID resolves a character ID and optional style to a VOICEVOX speaker ID.
// Falls back to the character's default style when style is empty or not found.
func (s *Synth) resolveSpeakerID(charID, style string) int {
	if s.Config == nil {
		return 0
	}
	ch, ok := s.Config.Characters[charID]
	if !ok {
		return 0
	}
	id, _ := ch.SpeakerID(style)
	return id
}
