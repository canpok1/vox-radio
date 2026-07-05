package synth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/logging"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/model"
)

const pollIntervalDefault = time.Second

// Synth synthesizes speech segments from a script using the VOICEVOX HTTP API.
// Clients is keyed by voicevox.engines name (or config.DefaultEngineName in
// url-only mode); segments are routed to an engine by their character's
// CharacterConfig.EffectiveEngine().
type Synth struct {
	Clients      map[string]VoicevoxClient
	urls         map[string]string
	Config       *config.Config
	getDuration  func(path string) (float64, error)
	logger       *slog.Logger
	pollInterval time.Duration
}

// Option configures a Synth.
type Option func(*Synth)

// WithLogger sets the logger used for progress messages.
func WithLogger(l *slog.Logger) Option {
	return func(s *Synth) { s.logger = l }
}

// New creates a new Synth with an HTTP VOICEVOX client per engine. urls maps
// engine name (config.DefaultEngineName in url-only mode) to its resolved
// URL, as returned by config.VoicevoxConfig.EffectiveURLs().
func New(urls map[string]string, cfg *config.Config, opts ...Option) *Synth {
	clients := make(map[string]VoicevoxClient, len(urls))
	for name, u := range urls {
		clients[name] = NewClient(u)
	}
	s := &Synth{
		Clients:      clients,
		urls:         urls,
		Config:       cfg,
		getDuration:  mediainfo.Duration,
		logger:       slog.Default(),
		pollInterval: pollIntervalDefault,
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

	var timeout time.Duration
	if s.Config != nil {
		timeout = s.Config.Voicevox.EffectiveStartupTimeout()
	}
	if err := waitForAllReady(ctx, s.readinessTargets(), timeout, s.pollInterval); err != nil {
		return nil, fmt.Errorf("wait for voicevox: %w", err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	speechSegs := make([]model.ScriptSegment, 0, len(script.Segments))
	for _, seg := range script.Segments {
		if seg.Type == model.SegmentTypeSpeech {
			speechSegs = append(speechSegs, seg)
		}
	}

	done := logging.StartStep(ctx, logger, fmt.Sprintf("開始 (%dクリップ)", len(speechSegs)))

	var presets config.VoicevoxPresets
	if s.Config != nil {
		presets = s.Config.Voicevox.EffectivePresets()
	}

	clips := make([]model.ClipMeta, 0, len(speechSegs))

	for i, seg := range speechSegs {
		logger.Info(fmt.Sprintf("クリップを合成中 (%d/%d)", i+1, len(speechSegs)))
		logger.Debug("クリップ詳細", "speaker", seg.SpeakerRole, "style", seg.Style, "text_chars", utf8.RuneCountInString(seg.Text))

		speakerID := s.resolveSpeakerID(seg.SpeakerRole, seg.Style)
		client := s.resolveClient(seg.SpeakerRole)
		clipFile := fmt.Sprintf("clip_%03d.wav", i)
		clipPath := filepath.Join(outDir, clipFile)

		if err := s.synthesize(ctx, client, seg, speakerID, clipPath, presets); err != nil {
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

	done(fmt.Sprintf("%dクリップ", len(clips)))

	return meta, nil
}

func (s *Synth) synthesize(ctx context.Context, client VoicevoxClient, seg model.ScriptSegment, speakerID int, outPath string, presets config.VoicevoxPresets) error {
	query, err := client.AudioQuery(ctx, seg.Text, speakerID)
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

	wavBytes, err := client.Synthesis(ctx, query, speakerID)
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

// resolveClient resolves a character ID to the VoicevoxClient for its
// CharacterConfig.EffectiveEngine(). Falls back to config.DefaultEngineName
// when the character is unknown or its engine has no matching client.
func (s *Synth) resolveClient(charID string) VoicevoxClient {
	name := config.DefaultEngineName
	if s.Config != nil {
		if ch, ok := s.Config.Characters[charID]; ok {
			name = ch.EffectiveEngine()
		}
	}
	if c, ok := s.Clients[name]; ok {
		return c
	}
	return s.Clients[config.DefaultEngineName]
}

// readinessTargets builds the readiness poll targets for every registered client.
func (s *Synth) readinessTargets() []readinessTarget {
	targets := make([]readinessTarget, 0, len(s.Clients))
	for name, client := range s.Clients {
		targets = append(targets, readinessTarget{name: name, url: s.urls[name], client: client})
	}
	return targets
}

// CheckReadiness verifies every VOICEVOX-compatible engine in urls (keyed by
// engine name, as returned by config.VoicevoxConfig.EffectiveURLs()) is
// reachable by polling its /version endpoint concurrently, waiting up to the
// configured startup timeout (config.VoicevoxConfig.EffectiveStartupTimeout;
// zero disables the check).
//
// It is intended to be called at command startup — before any LLM work — so an
// unreachable engine fails fast instead of only surfacing at the synth stage after
// expensive LLM steps have already run. Synth.Run keeps its own readiness wait as a
// defence for standalone use; when this ran first the engines are already ready so
// that later poll returns immediately.
func CheckReadiness(ctx context.Context, urls map[string]string, cfg *config.Config) error {
	var timeout time.Duration
	if cfg != nil {
		timeout = cfg.Voicevox.EffectiveStartupTimeout()
	}
	targets := make([]readinessTarget, 0, len(urls))
	for name, u := range urls {
		targets = append(targets, readinessTarget{name: name, url: u, client: NewClient(u)})
	}
	if err := waitForAllReady(ctx, targets, timeout, pollIntervalDefault); err != nil {
		return fmt.Errorf("VOICEVOX エンジンに接続できません: %w。VOICEVOX を起動してください（起動待機は voicevox.startup_timeout_seconds で調整、0 で無効）", err)
	}
	return nil
}

// readinessTarget identifies a single VOICEVOX-compatible engine for a readiness poll.
type readinessTarget struct {
	name   string
	url    string
	client VoicevoxClient
}

// waitForAllReady polls every target's /version endpoint concurrently until each
// responds successfully, waiting up to timeout. Returns nil immediately when
// timeout is zero (disabled) or there are no targets. On failure, returns a
// joined error naming the engine and URL for each unreachable target.
func waitForAllReady(ctx context.Context, targets []readinessTarget, timeout, interval time.Duration) error {
	if timeout == 0 || len(targets) == 0 {
		return nil
	}
	errCh := make(chan error, len(targets))
	for _, t := range targets {
		t := t
		go func() {
			err := waitForReady(ctx, t.client, timeout, interval)
			if err != nil {
				err = fmt.Errorf("%s (%s): %w", t.name, t.url, err)
			}
			errCh <- err
		}()
	}
	var errs []error
	for range targets {
		if err := <-errCh; err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// waitForReady polls the VOICEVOX /version endpoint until it responds successfully.
// Returns nil immediately when timeout is zero (disabled) or when the engine is ready.
// Returns an error if the context is cancelled or the timeout expires.
func waitForReady(ctx context.Context, client VoicevoxClient, timeout, interval time.Duration) error {
	if timeout == 0 {
		return nil
	}
	deadline := time.Now().Add(timeout)
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		if _, err := client.Version(ctx); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("voicevox did not become ready within %s", timeout)
		}
		timer.Reset(interval)
	}
}
