package model_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestArticle_Text_BodyPreferred(t *testing.T) {
	a := model.Article{
		DedupKey:    "k",
		Title:       "t",
		Description: "feed text",
		Body:        "html text",
	}
	if got := a.Text(); got != "html text" {
		t.Errorf("Text() = %q, want %q", got, "html text")
	}
}

func TestArticle_Text_DescriptionFallback(t *testing.T) {
	a := model.Article{
		DedupKey:    "k",
		Title:       "t",
		Description: "feed text",
	}
	if got := a.Text(); got != "feed text" {
		t.Errorf("Text() = %q, want %q", got, "feed text")
	}
}

func TestArticle_Text_BothEmpty(t *testing.T) {
	a := model.Article{DedupKey: "k", Title: "t"}
	if got := a.Text(); got != "" {
		t.Errorf("Text() = %q, want empty", got)
	}
}
