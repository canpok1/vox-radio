package feed

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"github.com/canpok1/vox-radio/internal/fileio"
)

const (
	DefaultPublicDir     = "public"
	DefaultCreditsHeader = "クレジット"
)

// FeedConfig holds RSS feed metadata for feed-spec.yaml.
type FeedConfig struct {
	Language         string `yaml:"language"`
	Email            string `yaml:"email"`
	Category         string `yaml:"category"`
	Explicit         bool   `yaml:"explicit"`
	CoverImageURL    string `yaml:"cover_image_url"`
	SiteURL          string `yaml:"site_url"`
	AudioURLTemplate string `yaml:"audio_url_template"`
	Credit           string `yaml:"credit"`
	CreditsHeader    string `yaml:"credits_header,omitempty"`
}

// EffectiveCreditsHeader returns the configured credits section header, or DefaultCreditsHeader if not set.
func (c FeedConfig) EffectiveCreditsHeader() string {
	if c.CreditsHeader == "" {
		return DefaultCreditsHeader
	}
	return c.CreditsHeader
}

// OutputConfig holds output settings for feed-spec.yaml.
type OutputConfig struct {
	Public string `yaml:"public"`
}

// FeedSpec is the top-level structure for feed-spec.yaml.
type FeedSpec struct {
	Feed   FeedConfig   `yaml:"feed"`
	Output OutputConfig `yaml:"output"`
}

// EffectivePublicDir returns the configured public directory or DefaultPublicDir if not set.
func (c FeedSpec) EffectivePublicDir() string {
	if c.Output.Public == "" {
		return DefaultPublicDir
	}
	return c.Output.Public
}

// LoadFeedSpec reads and parses a feed-spec.yaml file.
func LoadFeedSpec(path string) (FeedSpec, error) {
	return loadFeedSpecWith(path, false)
}

// LoadFeedSpecStrict reads and parses a feed-spec.yaml file with strict mode.
// Unknown keys in the YAML will cause an error (detects typos).
func LoadFeedSpecStrict(path string) (FeedSpec, error) {
	return loadFeedSpecWith(path, true)
}

func loadFeedSpecWith(path string, strict bool) (FeedSpec, error) {
	var cfg FeedSpec
	if err := fileio.DecodeYAML(path, &cfg, strict); err != nil {
		return FeedSpec{}, fmt.Errorf("load feed spec: %w", err)
	}
	return cfg, nil
}

// ValidateFeedSpec validates the semantic correctness of a FeedSpec.
// It checks required fields, URL/email formats, and audio_url_template placeholders.
// Multiple errors are collected and returned via errors.Join.
func ValidateFeedSpec(spec FeedSpec) error {
	var errs []error

	// (b) required fields
	if spec.Feed.Language == "" {
		errs = append(errs, errors.New("feed.language is required"))
	}
	if spec.Feed.Email == "" {
		errs = append(errs, errors.New("feed.email is required"))
	}
	if spec.Feed.SiteURL == "" {
		errs = append(errs, errors.New("feed.site_url is required"))
	}
	if spec.Feed.AudioURLTemplate == "" {
		errs = append(errs, errors.New("feed.audio_url_template is required"))
	}

	// (c) URL/email format checks (only when non-empty)
	if spec.Feed.Email != "" {
		if _, err := mail.ParseAddress(spec.Feed.Email); err != nil {
			errs = append(errs, fmt.Errorf("feed.email is invalid: %w", err))
		}
	}
	if spec.Feed.SiteURL != "" {
		if err := validateAbsoluteURL(spec.Feed.SiteURL); err != nil {
			errs = append(errs, fmt.Errorf("feed.site_url %w", err))
		}
	}
	if spec.Feed.AudioURLTemplate != "" {
		// (c) URL format: replace placeholders before validating
		expanded := strings.ReplaceAll(spec.Feed.AudioURLTemplate, "{episode_number}", "1")
		expanded = strings.ReplaceAll(expanded, "{audio_file}", "ep.mp3")
		if err := validateAbsoluteURL(expanded); err != nil {
			errs = append(errs, fmt.Errorf("feed.audio_url_template %w", err))
		}
		// (d) placeholder presence
		if !strings.Contains(spec.Feed.AudioURLTemplate, "{episode_number}") {
			errs = append(errs, errors.New("feed.audio_url_template must contain {episode_number}"))
		}
		if !strings.Contains(spec.Feed.AudioURLTemplate, "{audio_file}") {
			errs = append(errs, errors.New("feed.audio_url_template must contain {audio_file}"))
		}
	}
	if spec.Feed.CoverImageURL != "" {
		if err := validateAbsoluteURL(spec.Feed.CoverImageURL); err != nil {
			errs = append(errs, fmt.Errorf("feed.cover_image_url %w", err))
		}
	}

	return errors.Join(errs...)
}

func validateAbsoluteURL(s string) error {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return fmt.Errorf("must be a valid absolute URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("must use http or https scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("must have a host")
	}
	return nil
}
