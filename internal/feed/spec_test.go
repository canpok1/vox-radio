package feed_test

import (
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/feed"
	"github.com/canpok1/vox-radio/internal/testutil"
)

const validFeedSpecYAML = `
feed:
  language: ja
  email: test@example.com
  site_url: https://example.com/
  audio_url_template: "https://example.com/ep-{episode_number}/{audio_file}"
output:
  public: public
`

func TestLoadFeedSpec_ValidYAML(t *testing.T) {
	content := `
feed:
  language: ja
  email: test@example.com
  category: Technology
  explicit: false
  cover_image_url: https://example.com/cover.png
  site_url: https://example.com/
  audio_url_template: "https://github.com/owner/repo/releases/download/ep-{episode_number}/{audio_file}"
  credit: "VOICEVOX:ずんだもん"
output:
  public: public
`
	path := testutil.WriteTempFile(t, "feed-spec.yaml", []byte(content))

	cfg, err := feed.LoadFeedSpec(path)
	if err != nil {
		t.Fatalf("LoadFeedSpec: unexpected error: %v", err)
	}

	if cfg.Feed.Language != "ja" {
		t.Errorf("Feed.Language: got %q, want %q", cfg.Feed.Language, "ja")
	}
	if cfg.Feed.Email != "test@example.com" {
		t.Errorf("Feed.Email: got %q, want %q", cfg.Feed.Email, "test@example.com")
	}
	if cfg.Feed.Category != "Technology" {
		t.Errorf("Feed.Category: got %q, want %q", cfg.Feed.Category, "Technology")
	}
	if cfg.Feed.Explicit {
		t.Errorf("Feed.Explicit: got true, want false")
	}
	if cfg.Feed.CoverImageURL != "https://example.com/cover.png" {
		t.Errorf("Feed.CoverImageURL: got %q, want %q", cfg.Feed.CoverImageURL, "https://example.com/cover.png")
	}
	if cfg.Feed.SiteURL != "https://example.com/" {
		t.Errorf("Feed.SiteURL: got %q, want %q", cfg.Feed.SiteURL, "https://example.com/")
	}
	wantTemplate := "https://github.com/owner/repo/releases/download/ep-{episode_number}/{audio_file}"
	if cfg.Feed.AudioURLTemplate != wantTemplate {
		t.Errorf("Feed.AudioURLTemplate: got %q, want %q", cfg.Feed.AudioURLTemplate, wantTemplate)
	}
	if cfg.Feed.Credit != "VOICEVOX:ずんだもん" {
		t.Errorf("Feed.Credit: got %q, want %q", cfg.Feed.Credit, "VOICEVOX:ずんだもん")
	}
	if cfg.Output.Public != "public" {
		t.Errorf("Output.Public: got %q, want %q", cfg.Output.Public, "public")
	}
}

func TestLoadFeedSpec_FileNotExist(t *testing.T) {
	_, err := feed.LoadFeedSpec("/nonexistent/path/feed-spec.yaml")
	if err == nil {
		t.Error("LoadFeedSpec: expected error for nonexistent file, got nil")
	}
}

func TestLoadFeedSpec_CreditsHeaderField(t *testing.T) {
	content := `
feed:
  language: ja
  email: test@example.com
  site_url: https://example.com/
  audio_url_template: "https://example.com/ep-{episode_number}/{audio_file}"
  credits_header: Attributions
output:
  public: public
`
	path := testutil.WriteTempFile(t, "feed-spec.yaml", []byte(content))

	cfg, err := feed.LoadFeedSpec(path)
	if err != nil {
		t.Fatalf("LoadFeedSpec: unexpected error: %v", err)
	}
	if cfg.Feed.CreditsHeader != "Attributions" {
		t.Errorf("Feed.CreditsHeader: got %q, want %q", cfg.Feed.CreditsHeader, "Attributions")
	}
}

func TestFeedConfig_EffectiveCreditsHeader_DefaultWhenEmpty(t *testing.T) {
	cfg := feed.FeedConfig{}
	got := cfg.EffectiveCreditsHeader()
	if got != feed.DefaultCreditsHeader {
		t.Errorf("EffectiveCreditsHeader(): got %q, want %q", got, feed.DefaultCreditsHeader)
	}
}

func TestFeedConfig_EffectiveCreditsHeader_Custom(t *testing.T) {
	cfg := feed.FeedConfig{CreditsHeader: "Credits"}
	got := cfg.EffectiveCreditsHeader()
	if got != "Credits" {
		t.Errorf("EffectiveCreditsHeader(): got %q, want %q", got, "Credits")
	}
}

func TestFeedSpec_EffectivePublicDir_Default(t *testing.T) {
	cfg := feed.FeedSpec{}
	got := cfg.EffectivePublicDir()
	if got != feed.DefaultPublicDir {
		t.Errorf("EffectivePublicDir(): got %q, want %q", got, feed.DefaultPublicDir)
	}
}

func TestFeedSpec_EffectivePublicDir_Custom(t *testing.T) {
	cfg := feed.FeedSpec{
		Output: feed.OutputConfig{Public: "dist/public"},
	}
	got := cfg.EffectivePublicDir()
	if got != "dist/public" {
		t.Errorf("EffectivePublicDir(): got %q, want %q", got, "dist/public")
	}
}

func TestLoadFeedSpecStrict_Valid(t *testing.T) {
	path := testutil.WriteTempFile(t, "feed-spec.yaml", []byte(validFeedSpecYAML))
	_, err := feed.LoadFeedSpecStrict(path)
	if err != nil {
		t.Fatalf("LoadFeedSpecStrict: unexpected error: %v", err)
	}
}

func TestLoadFeedSpecStrict_UnknownKey(t *testing.T) {
	path := testutil.WriteTempFile(t, "feed-spec.yaml", []byte(validFeedSpecYAML+"\nunknown_field: value\n"))
	_, err := feed.LoadFeedSpecStrict(path)
	if err == nil {
		t.Error("LoadFeedSpecStrict: expected error for unknown key, got nil")
	}
}

// program_id は FeedSpec から削除されたため、strict モードで unknown key エラーになること
func TestLoadFeedSpecStrict_ProgramIDField_RaisesUnknownKey(t *testing.T) {
	content := "program_id: my-radio\n" + validFeedSpecYAML
	path := testutil.WriteTempFile(t, "feed-spec.yaml", []byte(content))
	_, err := feed.LoadFeedSpecStrict(path)
	if err == nil {
		t.Error("LoadFeedSpecStrict: expected error for program_id (unknown key), got nil")
	}
}

func TestLoadFeedSpecStrict_FileNotExist(t *testing.T) {
	_, err := feed.LoadFeedSpecStrict("/nonexistent/path/feed-spec.yaml")
	if err == nil {
		t.Error("LoadFeedSpecStrict: expected error for nonexistent file, got nil")
	}
}

func validSpec() feed.FeedSpec {
	return feed.FeedSpec{
		Feed: feed.FeedConfig{
			Language:         "ja",
			Email:            "test@example.com",
			SiteURL:          "https://example.com/",
			AudioURLTemplate: "https://example.com/ep-{episode_number}/{audio_file}",
		},
	}
}

func TestValidateFeedSpec(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(s *feed.FeedSpec)
		wantErr     bool
		errContains []string
	}{
		{
			name:    "valid full spec",
			mutate:  nil,
			wantErr: false,
		},
		{
			name: "optional fields empty",
			mutate: func(s *feed.FeedSpec) {
				s.Feed.Category = ""
				s.Feed.CoverImageURL = ""
				s.Feed.Credit = ""
				s.Output.Public = ""
			},
			wantErr: false,
		},
		{
			name:        "missing language",
			mutate:      func(s *feed.FeedSpec) { s.Feed.Language = "" },
			wantErr:     true,
			errContains: []string{"feed.language"},
		},
		{
			name:        "missing email",
			mutate:      func(s *feed.FeedSpec) { s.Feed.Email = "" },
			wantErr:     true,
			errContains: []string{"feed.email"},
		},
		{
			name:        "missing site_url",
			mutate:      func(s *feed.FeedSpec) { s.Feed.SiteURL = "" },
			wantErr:     true,
			errContains: []string{"feed.site_url"},
		},
		{
			name:        "missing audio_url_template",
			mutate:      func(s *feed.FeedSpec) { s.Feed.AudioURLTemplate = "" },
			wantErr:     true,
			errContains: []string{"feed.audio_url_template"},
		},
		{
			name:        "invalid email format",
			mutate:      func(s *feed.FeedSpec) { s.Feed.Email = "not-an-email" },
			wantErr:     true,
			errContains: []string{"feed.email"},
		},
		{
			name:        "invalid site_url (relative)",
			mutate:      func(s *feed.FeedSpec) { s.Feed.SiteURL = "/relative/path" },
			wantErr:     true,
			errContains: []string{"feed.site_url"},
		},
		{
			name:        "invalid site_url (no scheme)",
			mutate:      func(s *feed.FeedSpec) { s.Feed.SiteURL = "example.com" },
			wantErr:     true,
			errContains: []string{"feed.site_url"},
		},
		{
			name:        "invalid audio_url_template (relative)",
			mutate:      func(s *feed.FeedSpec) { s.Feed.AudioURLTemplate = "/ep-{episode_number}/{audio_file}" },
			wantErr:     true,
			errContains: []string{"feed.audio_url_template"},
		},
		{
			name:        "cover_image_url invalid when non-empty",
			mutate:      func(s *feed.FeedSpec) { s.Feed.CoverImageURL = "not-a-url" },
			wantErr:     true,
			errContains: []string{"feed.cover_image_url"},
		},
		{
			name:        "audio_url_template missing episode_number placeholder",
			mutate:      func(s *feed.FeedSpec) { s.Feed.AudioURLTemplate = "https://example.com/{audio_file}" },
			wantErr:     true,
			errContains: []string{"episode_number"},
		},
		{
			name:        "audio_url_template missing audio_file placeholder",
			mutate:      func(s *feed.FeedSpec) { s.Feed.AudioURLTemplate = "https://example.com/{episode_number}" },
			wantErr:     true,
			errContains: []string{"audio_file"},
		},
		{
			name: "multiple errors collected",
			mutate: func(s *feed.FeedSpec) {
				s.Feed.Language = ""
				s.Feed.Email = ""
			},
			wantErr:     true,
			errContains: []string{"feed.language", "feed.email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := validSpec()
			if tt.mutate != nil {
				tt.mutate(&spec)
			}
			err := feed.ValidateFeedSpec(spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ValidateFeedSpec: expected error, got nil")
				}
				for _, want := range tt.errContains {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error should contain %q, got: %v", want, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFeedSpec: unexpected error: %v", err)
				}
			}
		})
	}
}
