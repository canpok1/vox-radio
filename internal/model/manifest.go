package model

// ArticleRef is a reference to an article included in a manifest corner.
type ArticleRef struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ManifestCorner represents a corner in the manifest with its articles.
type ManifestCorner struct {
	Title    string       `json:"title"`
	Articles []ArticleRef `json:"articles"`
}

// Manifest is the content manifest output alongside an mp3 episode.
type Manifest struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Datetime    string           `json:"datetime"`
	AudioFile   string           `json:"audio_file"`
	Corners     []ManifestCorner `json:"corners"`
}
