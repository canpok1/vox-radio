package feed

import (
	"encoding/xml"
	"fmt"
	"strconv"
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
	ItunesAuthor   string          `xml:"itunes:author,omitempty"`
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

type channelMeta struct {
	title       string
	description string
	author      string
}

func extractChannelMeta(entries []cache.Entry) channelMeta {
	e := cache.Last(entries)
	if e == nil {
		return channelMeta{}
	}
	return channelMeta{
		title:       e.Title,
		description: e.Description,
		author:      e.Author,
	}
}

// BuildFeed generates a podcast RSS 2.0 + iTunes feed XML from cache entries.
// Channel title/description/author come from the latest entry. Items are ordered newest first.
func BuildFeed(cfg FeedSpec, entries []cache.Entry) (string, error) {
	meta := extractChannelMeta(entries)

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
			Description:    itemDescription(cfg.Feed.EffectiveCreditsHeader(), e),
			GUID:           rssGUID{IsPermaLink: "false", Value: "ep-" + strconv.Itoa(e.EpisodeNumber)},
			PubDate:        pubDate(e.Datetime),
			Enclosure:      rssEnclosure{URL: url, Length: e.Bytes, Type: "audio/mpeg"},
			ItunesDuration: strconv.Itoa(e.DurationSec),
		}
		items = append(items, item)
	}

	root := rssRoot{
		Version:     "2.0",
		XmlnsItunes: "http://www.itunes.com/dtds/podcast-1.0.dtd",
		Channel: rssChannel{
			Title:          meta.title,
			Description:    meta.description,
			Language:       cfg.Feed.Language,
			Link:           cfg.Feed.SiteURL,
			ItunesAuthor:   meta.author,
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

// itemDescription builds the RSS item description by appending a credits section
// to the episode summary when credits are present.
func itemDescription(creditsHeader string, e cache.Entry) string {
	if len(e.Credits) == 0 {
		return e.Summary
	}
	return e.Summary + "\n\n" + creditsHeader + "\n" + strings.Join(e.Credits, "\n")
}

func audioURL(tmpl string, episodeNumber int, audioFile string) string {
	s := strings.ReplaceAll(tmpl, "{episode_number}", strconv.Itoa(episodeNumber))
	return strings.ReplaceAll(s, "{audio_file}", audioFile)
}

func itemTitle(e cache.Entry) string {
	return model.EpisodeDisplayTitle(e.EpisodeNumber, e.EpisodeTitle, e.Title)
}

func pubDate(datetime string) string {
	t, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		return datetime
	}
	return t.UTC().Format(time.RFC1123Z)
}
