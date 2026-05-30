package feed

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"text/template"
	"time"

	"github.com/canpok1/vox-radio/internal/config"
	"github.com/canpok1/vox-radio/internal/model"
)

const rfc1123Z = "Mon, 02 Jan 2006 15:04:05 -0700"

const feedTmplStr = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>{{x .Title}}</title>
    <description>{{x .Description}}</description>
    <language>{{x .Language}}</language>
    <link>{{x .Link}}</link>
    <itunes:author>{{x .Author}}</itunes:author>
    <itunes:category text="{{x .Category}}"/>
    <itunes:explicit>{{x .Explicit}}</itunes:explicit>
    <itunes:image href="{{x .CoverImageURL}}"/>
{{- range .Items}}
    <item>
      <guid isPermaLink="false">{{x .GUID}}</guid>
      <title>{{x .Title}}</title>
      <description>{{x .Description}}</description>
      <pubDate>{{x .PubDate}}</pubDate>
      <enclosure url="{{x .AudioURL}}" length="{{.Bytes}}" type="audio/mpeg"/>
      <itunes:duration>{{x .Duration}}</itunes:duration>
    </item>
{{- end}}
  </channel>
</rss>`

var feedTmpl = template.Must(template.New("feed").Funcs(template.FuncMap{
	"x": xmlEscape,
}).Parse(feedTmplStr))

type channelData struct {
	Title         string
	Description   string
	Language      string
	Link          string
	Author        string
	Category      string
	Explicit      string
	CoverImageURL string
	Items         []itemData
}

type itemData struct {
	GUID        string
	Title       string
	Description string
	PubDate     string
	AudioURL    string
	Bytes       int64
	Duration    string
}

func Generate(cfg config.PodcastConfig, episodes model.Episodes) ([]byte, error) {
	items := make([]itemData, len(episodes.Episodes))
	for i, ep := range episodes.Episodes {
		pubDate, err := formatPubDate(ep.PubDate)
		if err != nil {
			return nil, fmt.Errorf("episode %s: format pubDate: %w", ep.GUID, err)
		}
		items[i] = itemData{
			GUID:        ep.GUID,
			Title:       ep.Title,
			Description: ep.Description,
			PubDate:     pubDate,
			AudioURL:    ep.AudioURL,
			Bytes:       ep.Bytes,
			Duration:    ep.Duration,
		}
	}

	explicit := "no"
	if cfg.Explicit {
		explicit = "yes"
	}

	data := channelData{
		Title:         cfg.Title,
		Description:   cfg.Description,
		Language:      cfg.Language,
		Link:          cfg.SiteURL,
		Author:        cfg.Author,
		Category:      cfg.Category,
		Explicit:      explicit,
		CoverImageURL: cfg.CoverImageURL,
		Items:         items,
	}

	var buf bytes.Buffer
	if err := feedTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

func formatPubDate(pubDate string) (string, error) {
	t, err := time.Parse(time.RFC3339, pubDate)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", pubDate, err)
	}
	return t.UTC().Format(rfc1123Z), nil
}

func xmlEscape(s string) (string, error) {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return "", err
	}
	return buf.String(), nil
}
