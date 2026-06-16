package manifest

import (
	"time"
	"unicode/utf8"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

// BuildParams holds all parameters required to build a Manifest.
type BuildParams struct {
	Program           config.ProgramConfig
	Corners           []config.CornerConfig
	Rundown           model.Rundown
	AudioFile         string
	GeneratedAt       time.Time
	Summary           string
	CornerSummaries   map[string]model.CornerSummary
	ConversationNotes []model.ConversationNote
	EpisodeNumber     int
	EpisodeTitle      string
	Assets            config.AssetsConfig
	Characters        map[string]config.CharacterConfig
	Lines             *model.ScriptLines
	Script            *model.Script
	Clips             *model.ClipsMeta   // optional; used to compute SpeechSec and CharCount per corner
	CornerDurations   map[string]float64 // optional; keyed by CornerID, sets DurationSec per corner
}

// cornerClipStats computes per-corner speech duration and character count from ClipsMeta.
func cornerClipStats(clips *model.ClipsMeta) (speechSec map[string]float64, charCount map[string]int) {
	speechSec = make(map[string]float64)
	charCount = make(map[string]int)
	if clips == nil {
		return
	}
	for _, clip := range clips.Clips {
		speechSec[clip.CornerID] += clip.DurationSec
		charCount[clip.CornerID] += utf8.RuneCountInString(clip.Text)
	}
	return
}

// Build constructs a Manifest from BuildParams.
// BuildParams.CornerSummaries maps corner title to its LLM-generated summary; nil means no corner summaries.
// BuildParams.ConversationNotes contains non-rundown conversation information extracted from the episode.
// BuildParams.EpisodeNumber 0 means unknown (omitted from manifest). BuildParams.EpisodeTitle "" means unknown (omitted).
// Credits are collected internally from Assets/Characters/Lines/Script/Rundown.Casts.
func Build(p BuildParams) model.Manifest {
	credits := CollectCredits(CreditParams{
		Assets:     p.Assets,
		Characters: p.Characters,
		Lines:      p.Lines,
		Script:     p.Script,
		Casts:      p.Rundown.Casts,
	})
	cornerSpeechSec, cornerCharCount := cornerClipStats(p.Clips)
	cornerMap := p.Rundown.CornerMap()
	manifestCorners := make([]model.ManifestCorner, 0, len(p.Corners))
	for _, c := range p.Corners {
		rc := cornerMap[c.ID]
		refs := make([]model.ArticleRef, 0, len(rc.Articles))
		for _, a := range rc.Articles {
			refs = append(refs, model.ArticleRef{DedupKey: a.DedupKey, Title: a.Title, URL: a.URL})
		}
		cs := p.CornerSummaries[c.Title]
		mc := model.NewManifestCorner(c.ID, c.Title, cs.Summary, cs.Points, refs)
		mc.TargetSec = c.LengthSec
		mc.SpeechSec = cornerSpeechSec[c.ID]
		mc.DurationSec = p.CornerDurations[c.ID]
		mc.CharCount = cornerCharCount[c.ID]
		manifestCorners = append(manifestCorners, mc)
	}
	return model.Manifest{
		Title:             p.Program.Title,
		EpisodeNumber:     p.EpisodeNumber,
		EpisodeTitle:      p.EpisodeTitle,
		Author:            p.Program.Author,
		Description:       p.Program.Description,
		Summary:           p.Summary,
		Datetime:          p.GeneratedAt.UTC().Format(time.RFC3339),
		AudioFile:         p.AudioFile,
		Corners:           manifestCorners,
		ConversationNotes: model.NonNil(p.ConversationNotes),
		Casts:             model.NonNil(p.Rundown.Casts),
		Credits:           model.NonNil(credits),
	}
}
