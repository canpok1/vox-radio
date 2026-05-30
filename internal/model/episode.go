package model

type Episode struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PubDate     string `json:"pub_date"`
	AudioURL    string `json:"audio_url"`
	Bytes       int64  `json:"bytes"`
	Duration    string `json:"duration"`
}

type Episodes struct {
	Episodes []Episode `json:"episodes"`
}
