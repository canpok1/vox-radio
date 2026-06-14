package config

import (
	"fmt"
	"slices"
)

const (
	// CastTypeRegular は毎回または条件付きで出演するレギュラーキャストを表す。
	CastTypeRegular = "regular"
	// CastTypeGuest は条件付きで出演するゲストキャストを表す。
	CastTypeGuest = "guest"
)

// EpisodeCondition は回番号ベースの出現条件。コーナーとゲストで共有する。
type EpisodeCondition struct {
	Episodes []int             `yaml:"episodes,omitempty"` // この回番号で採用（明示リスト）
	Every    int               `yaml:"every,omitempty"`    // N の倍数回で採用（0 で無効）
	Offset   int               `yaml:"offset,omitempty"`   // every との剰余。未指定=0 で倍数回（episodeNumber%Every==Offset で採用）
	Not      *EpisodeCondition `yaml:"not,omitempty"`      // この条件に合致する回は除外（補集合）
}

// Matches は episodeNumber が条件に合致するか判定する。
func (c EpisodeCondition) Matches(episodeNumber int) bool {
	if episodeNumber <= 0 {
		return false
	}
	if c.Not != nil && c.Not.Matches(episodeNumber) {
		return false
	}
	// 肯定条件が両方未指定 → 全回が対象（not 単独で補集合を表現できるようにする）
	if len(c.Episodes) == 0 && c.Every == 0 {
		return true
	}
	return slices.Contains(c.Episodes, episodeNumber) ||
		(c.Every > 0 && episodeNumber%c.Every == c.Offset)
}

// CastConfig はキャスト1人分の設定（キャラIDは map のキーで持つため持たない）。
type CastConfig struct {
	Type      string            `yaml:"type"`                // "regular" | "guest"
	Role      string            `yaml:"role"`                // 番組全体での役割
	Condition *EpisodeCondition `yaml:"condition,omitempty"` // regular: 省略=毎回、guest: 必須
}

// EpisodeSpec holds episode-specific settings (program, corners, assets).
// It is loaded from episode-spec.yaml.
// Data sources (feeds, articles) are defined per-corner in corners[].source.
// Assets are loaded from the files listed in AssetsFiles and merged into Assets.
type EpisodeSpec struct {
	Program     ProgramConfig         `yaml:"program"`
	Corners     []CornerConfig        `yaml:"corners"`
	Casts       map[string]CastConfig `yaml:"casts,omitempty"` // 出演者名簿（旧 Guests を置換）
	AssetsFiles []string              `yaml:"assets_files"`
	Assets      AssetsConfig          `yaml:"-"`
}

// validateEpisodeCondition は EpisodeCondition の値が有効かを検証する共通ヘルパー。
// prefix はエラーメッセージの先頭に付加するフィールドパス文字列。
func validateEpisodeCondition(cond EpisodeCondition, prefix string) error {
	for _, e := range cond.Episodes {
		if e < 1 {
			return fmt.Errorf("%s.episodes: value %d must be >= 1", prefix, e)
		}
	}
	if cond.Every < 0 {
		return fmt.Errorf("%s.every: value %d must be >= 1", prefix, cond.Every)
	}
	if cond.Offset < 0 {
		return fmt.Errorf("%s.offset: value %d must be >= 0", prefix, cond.Offset)
	}
	if cond.Offset > 0 && cond.Every == 0 {
		return fmt.Errorf("%s.offset: requires every to be set", prefix)
	}
	if cond.Every > 0 && cond.Offset >= cond.Every {
		return fmt.Errorf("%s.offset: value %d must be < every (%d)", prefix, cond.Offset, cond.Every)
	}
	if len(cond.Episodes) == 0 && cond.Every == 0 && cond.Not == nil {
		return fmt.Errorf("%s: at least one of episodes, every, or not must be set", prefix)
	}
	if cond.Not != nil {
		if err := validateEpisodeCondition(*cond.Not, prefix+".not"); err != nil {
			return err
		}
	}
	return nil
}

// ValidateCasts checks that every cast character ID exists in chars,
// that type is valid, and that condition is set correctly (guest requires condition).
func (p *EpisodeSpec) ValidateCasts(chars map[string]CharacterConfig) error {
	for charID, c := range p.Casts {
		if _, ok := chars[charID]; !ok {
			return fmt.Errorf("casts[%q]: character not found in characters catalog", charID)
		}
		if c.Type != CastTypeRegular && c.Type != CastTypeGuest {
			return fmt.Errorf("casts[%q].type: must be %q or %q, got %q", charID, CastTypeRegular, CastTypeGuest, c.Type)
		}
		if c.Type == CastTypeGuest && c.Condition == nil {
			return fmt.Errorf("casts[%q].condition: required for guest type", charID)
		}
		if c.Condition != nil {
			if err := validateEpisodeCondition(*c.Condition, fmt.Sprintf("casts[%q].condition", charID)); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateCorners は corners の id・出現条件を検証する（キャラ不要・spec 内部整合のみ）。
// id は必須かつ番組内で一意。title もユーザー向け表示用に重複を禁止する。
func (p *EpisodeSpec) ValidateCorners() error {
	seenID := make(map[string]bool, len(p.Corners))
	seenTitle := make(map[string]bool, len(p.Corners))
	for i, c := range p.Corners {
		if c.ID == "" {
			return fmt.Errorf("corners[%d]: id is required", i)
		}
		if seenID[c.ID] {
			return fmt.Errorf("corners[%d]: id %q is duplicated", i, c.ID)
		}
		seenID[c.ID] = true
		if seenTitle[c.Title] {
			return fmt.Errorf("corners[%d]: title %q is duplicated", i, c.Title)
		}
		seenTitle[c.Title] = true
		if c.Condition != nil {
			if err := validateEpisodeCondition(*c.Condition, fmt.Sprintf("corners[%d].condition", i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// CornerSummaryLength returns the effective summary length (chars) for the corner matching title.
// Falls back to DefaultCornerSummaryLength when the corner is not found or summary_length is unset.
func (p *EpisodeSpec) CornerSummaryLength(title string) int {
	for _, c := range p.Corners {
		if c.Title == title {
			return c.EffectiveSummaryLength()
		}
	}
	return DefaultCornerSummaryLength
}

// ValidateProgram checks that program.id is set and audio_quality (if set) is a valid preset.
// program.id is the cache key (episodes are stored per program.id), so it is required.
func (p *EpisodeSpec) ValidateProgram() error {
	if p.Program.ID == "" {
		return fmt.Errorf("program.id is required (it is the cache key for episode history)")
	}
	if q := p.Program.EffectiveAudioQuality(); !slices.Contains(ValidAudioQualityPresets, q) {
		return fmt.Errorf("program.audio_quality: invalid preset %q (must be high, standard, or low)", p.Program.AudioQuality)
	}
	return nil
}

// ValidateCast checks that every character ID in corners[].cast is declared in casts.
// This ensures corner-only characters are forbidden, preventing rest-state leaks and typos.
func (p *EpisodeSpec) ValidateCast() error {
	for _, corner := range p.Corners {
		for charID := range corner.Cast {
			if _, ok := p.Casts[charID]; !ok {
				return fmt.Errorf("corners[%q].cast: character %q is not declared in casts", corner.Title, charID)
			}
		}
	}
	return nil
}

// ValidateAssets checks that corner-level audio/bgm keys reference existing assets.
func (p *EpisodeSpec) ValidateAssets() error {
	for _, corner := range p.Corners {
		if corner.StartAudio != nil {
			if err := validateAudioRef(corner.Title, "start_audio", corner.StartAudio, &p.Assets); err != nil {
				return err
			}
		}
		if corner.EndAudio != nil {
			if err := validateAudioRef(corner.Title, "end_audio", corner.EndAudio, &p.Assets); err != nil {
				return err
			}
		}
		if corner.BGM != "" {
			if _, ok := p.Assets.BGM[corner.BGM]; !ok {
				return fmt.Errorf("corners[%q].bgm: unknown bgm key %q", corner.Title, corner.BGM)
			}
		}
	}
	return nil
}

// Validate はすべてのバリデーションを実行する単一エントリポイント。
// Program・Corners・Cast・Assets・Casts の順に検証し、最初のエラーを返す。
func (p *EpisodeSpec) Validate(chars map[string]CharacterConfig) error {
	if err := p.ValidateProgram(); err != nil {
		return err
	}
	if err := p.ValidateCorners(); err != nil {
		return err
	}
	if err := p.ValidateCast(); err != nil {
		return err
	}
	if err := p.ValidateAssets(); err != nil {
		return err
	}
	return p.ValidateCasts(chars)
}

func validateAudioRef(cornerTitle, field string, ref *AudioRef, assets *AssetsConfig) error {
	switch ref.Type {
	case "jingle":
		if _, ok := assets.Jingle[ref.ID]; !ok {
			return fmt.Errorf("corners[%q].%s: unknown jingle key %q", cornerTitle, field, ref.ID)
		}
	case "se":
		if _, ok := assets.SE[ref.ID]; !ok {
			return fmt.Errorf("corners[%q].%s: unknown se key %q", cornerTitle, field, ref.ID)
		}
	default:
		return fmt.Errorf("corners[%q].%s: unknown type %q (must be jingle or se)", cornerTitle, field, ref.Type)
	}
	return nil
}
