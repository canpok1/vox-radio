package publish

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/mediainfo"
	"github.com/canpok1/vox-radio/internal/model"
	"github.com/canpok1/vox-radio/internal/publish/feed"
	"github.com/canpok1/vox-radio/internal/publish/hosting"
)

// Publisher orchestrates the publish pipeline.
type Publisher struct {
	Hosting     hosting.Hosting
	Podcast     config.PodcastConfig
	getDuration func(path string) (float64, error)
	getFileSize func(path string) (int64, error)
}

// Options holds optional parameters for a publish run.
type Options struct {
	Date        string // YYYY-MM-DD; defaults to today
	Title       string // defaults to "<date> <podcast.title>"
	Description string
}

// New creates a Publisher using ffprobe for duration detection.
func New(h hosting.Hosting, podcast config.PodcastConfig) *Publisher {
	return &Publisher{
		Hosting:     h,
		Podcast:     podcast,
		getDuration: mediainfo.Duration,
		getFileSize: mediainfo.FileSize,
	}
}

// Run executes the publish pipeline: upload mp3, update episodes.json, generate and upload feed.xml.
func (p *Publisher) Run(ctx context.Context, mp3Path string, opts Options) error {
	date := opts.Date
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	pubTime, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("parse date %q: %w", date, err)
	}

	title := opts.Title
	if title == "" {
		title = date + " " + p.Podcast.Title
	}

	fileBytes, err := p.getFileSize(mp3Path)
	if err != nil {
		return fmt.Errorf("get file size: %w", err)
	}

	durSec, err := p.getDuration(mp3Path)
	if err != nil {
		return fmt.Errorf("get duration: %w", err)
	}

	f, err := os.Open(mp3Path)
	if err != nil {
		return fmt.Errorf("open mp3: %w", err)
	}
	defer func() { _ = f.Close() }()

	audioName := "episode_" + date + ".mp3"
	audioURL, err := p.Hosting.PutAudio(ctx, audioName, f)
	if err != nil {
		return fmt.Errorf("put audio: %w", err)
	}

	episodes, err := p.Hosting.LoadEpisodes(ctx)
	if err != nil {
		return fmt.Errorf("load episodes: %w", err)
	}

	newEp := model.Episode{
		GUID:        "episode-" + date,
		Title:       title,
		Description: opts.Description,
		PubDate:     pubTime.Format(time.RFC3339),
		AudioURL:    audioURL,
		Bytes:       fileBytes,
		Duration:    formatDuration(durSec),
	}

	episodes.Episodes = append([]model.Episode{newEp}, episodes.Episodes...)

	if p.Podcast.MaxItems > 0 && len(episodes.Episodes) > p.Podcast.MaxItems {
		episodes.Episodes = episodes.Episodes[:p.Podcast.MaxItems]
	}

	if err := p.Hosting.SaveEpisodes(ctx, episodes); err != nil {
		return fmt.Errorf("save episodes: %w", err)
	}

	feedXML, err := feed.Generate(p.Podcast, episodes)
	if err != nil {
		return fmt.Errorf("generate feed: %w", err)
	}

	if _, err := p.Hosting.PutFeed(ctx, feedXML); err != nil {
		return fmt.Errorf("put feed: %w", err)
	}

	return nil
}

func formatDuration(sec float64) string {
	total := int(sec)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
