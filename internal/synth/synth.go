package synth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/model"
)

// Synth synthesizes speech segments from a script using the VOICEVOX HTTP API
type Synth struct {
	Client      VoicevoxClient
	Config      *config.Config
	getDuration func(path string) (float64, error)
}

// New creates a new Synth with an HTTP VOICEVOX client.
func New(engineURL string, cfg *config.Config) *Synth {
	return &Synth{
		Client:      NewClient(engineURL),
		Config:      cfg,
		getDuration: mediainfo.Duration,
	}
}

// Run synthesizes all speech segments and saves clip_NNN.wav files to outDir.
// It also writes clips.json with metadata including durations.
func (s *Synth) Run(ctx context.Context, script model.Script, outDir string) (*model.ClipsMeta, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	clips := make([]model.ClipMeta, 0)
	clipIdx := 0

	for _, seg := range script.Segments {
		if seg.Type != model.SegmentTypeSpeech {
			continue
		}

		speakerID := s.resolveSpeakerID(seg.SpeakerRole, seg.Style)
		clipFile := fmt.Sprintf("clip_%03d.wav", clipIdx)
		clipPath := filepath.Join(outDir, clipFile)

		if err := s.synthesize(ctx, seg.Text, speakerID, clipPath); err != nil {
			return nil, fmt.Errorf("synthesize clip %d: %w", clipIdx, err)
		}

		dur, err := s.getDuration(clipPath)
		if err != nil {
			return nil, fmt.Errorf("get duration of %s: %w", clipFile, err)
		}

		clips = append(clips, model.ClipMeta{
			Index:       clipIdx,
			File:        clipFile,
			DurationSec: dur,
			SpeakerRole: seg.SpeakerRole,
			Style:       seg.Style,
			Text:        seg.Text,
		})
		clipIdx++
	}

	meta := &model.ClipsMeta{Clips: clips}

	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal clips meta: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "clips.json"), b, 0o644); err != nil {
		return nil, fmt.Errorf("write clips.json: %w", err)
	}

	return meta, nil
}

func (s *Synth) synthesize(ctx context.Context, text string, speakerID int, outPath string) error {
	query, err := s.Client.AudioQuery(ctx, text, speakerID)
	if err != nil {
		return fmt.Errorf("audio query: %w", err)
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
