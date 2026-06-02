package feed

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/canpok1/vox-radio/internal/cache"
	"github.com/canpok1/vox-radio/internal/model"
)

type rssRoot struct {
	XMLName     xml.Name   `xml:"rss"`
	Version     string     `xml:"version,attr"`
	XmlnsItunes string     `xml:"xmlns:itunes,attr"`
	Channel     rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title          string          `xml:"title"`
	Description    string          `xml:"description"`
	Language       string          `xml:"language"`
	Link           string          `xml:"link"`
	ItunesAuthor   string          `xml:"itunes:author"`
	ItunesEmail    string          `xml:"itunes:email"`
	ItunesCategory *itunesCategory `xml:"itunes:category"`
	ItunesExplicit string          `xml:"itunes:explicit"`
	ItunesImage    *itunesImage    `xml:"itunes:image"`
	Items          []rssItem       `xml:"item"`
}

type itunesCategory struct {
	Text string `xml:"text,attr"`
}

type itunesImage struct {
	Href string `xml:"href,attr"`
}

type rssItem struct {
	Title          string       `xml:"title"`
	Description    string       `xml:"description"`
	GUID           rssGUID      `xml:"guid"`
	PubDate        string       `xml:"pubDate"`
	Enclosure      rssEnclosure `xml:"enclosure"`
	ItunesDuration string       `xml:"itunes:duration"`
	ItunesAuthor   string       `xml:"itunes:author,omitempty"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// BuildFeed generates a podcast RSS 2.0 + iTunes feed XML from cache entries.
// Channel title/description come from the latest entry. Items are ordered newest first.
func BuildFeed(cfg model.DistributionConfig, entries []cache.Entry) (string, error) {
	var channelTitle, channelDescription string
	if len(entries) > 0 {
		latest := entries[len(entries)-1]
		channelTitle = latest.Title
		channelDescription = latest.Description
	}

	var cat *itunesCategory
	if cfg.Feed.Category != "" {
		cat = &itunesCategory{Text: cfg.Feed.Category}
	}
	var img *itunesImage
	if cfg.Feed.CoverImageURL != "" {
		img = &itunesImage{Href: cfg.Feed.CoverImageURL}
	}

	explicitStr := "false"
	if cfg.Feed.Explicit {
		explicitStr = "true"
	}

	items := make([]rssItem, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		url := audioURL(cfg.Feed.AudioURLTemplate, e.EpisodeNumber, e.AudioFile)
		item := rssItem{
			Title:          itemTitle(e),
			Description:    e.Summary,
			GUID:           rssGUID{IsPermaLink: "false", Value: fmt.Sprintf("ep-%d", e.EpisodeNumber)},
			PubDate:        pubDate(e.Datetime),
			Enclosure:      rssEnclosure{URL: url, Length: e.Bytes, Type: "audio/mpeg"},
			ItunesDuration: fmt.Sprintf("%d", e.DurationSec),
		}
		if cfg.Feed.Credit != "" {
			item.ItunesAuthor = cfg.Feed.Credit
		}
		items = append(items, item)
	}

	root := rssRoot{
		Version:     "2.0",
		XmlnsItunes: "http://www.itunes.com/dtds/podcast-1.0.dtd",
		Channel: rssChannel{
			Title:          channelTitle,
			Description:    channelDescription,
			Language:       cfg.Feed.Language,
			Link:           cfg.Feed.SiteURL,
			ItunesAuthor:   cfg.Feed.Author,
			ItunesEmail:    cfg.Feed.Email,
			ItunesCategory: cat,
			ItunesExplicit: explicitStr,
			ItunesImage:    img,
			Items:          items,
		},
	}

	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal feed xml: %w", err)
	}
	return xml.Header + string(out) + "\n", nil
}

func audioURL(tmpl string, episodeNumber int, audioFile string) string {
	r := strings.NewReplacer(
		"{episode_number}", fmt.Sprintf("%d", episodeNumber),
		"{audio_file}", audioFile,
	)
	return r.Replace(tmpl)
}

func itemTitle(e cache.Entry) string {
	if e.EpisodeNumber > 0 && e.EpisodeTitle != "" {
		return fmt.Sprintf("第%d回 %s", e.EpisodeNumber, e.EpisodeTitle)
	}
	if e.EpisodeNumber > 0 {
		return fmt.Sprintf("第%d回", e.EpisodeNumber)
	}
	return e.Title
}

func pubDate(datetime string) string {
	t, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		return datetime
	}
	return t.UTC().Format(time.RFC1123Z)
}
