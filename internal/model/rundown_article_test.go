package model_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestRundownArticle_Text_BodyPreferred(t *testing.T) {
	a := model.RundownArticle{
		DedupKey:    "k",
		Title:       "t",
		Description: "feed text",
		Body:        "html text",
		Points:      []string{},
	}
	if got := a.Text(); got != "html text" {
		t.Errorf("Text() = %q, want %q", got, "html text")
	}
}

func TestRundownArticle_Text_DescriptionFallback(t *testing.T) {
	a := model.RundownArticle{
		DedupKey:    "k",
		Title:       "t",
		Description: "feed text",
		Points:      []string{},
	}
	if got := a.Text(); got != "feed text" {
		t.Errorf("Text() = %q, want %q", got, "feed text")
	}
}

func TestRundownArticle_Text_BothEmpty(t *testing.T) {
	a := model.RundownArticle{DedupKey: "k", Title: "t", Points: []string{}}
	if got := a.Text(); got != "" {
		t.Errorf("Text() = %q, want empty", got)
	}
}

func TestNewRundownArticle_DescriptionAndBody(t *testing.T) {
	a := model.NewRundownArticle("k", "url", "title", "desc", "body", []string{"p"}, "src", "auth", "pub")
	if a.Description != "desc" {
		t.Errorf("Description = %q, want %q", a.Description, "desc")
	}
	if a.Body != "body" {
		t.Errorf("Body = %q, want %q", a.Body, "body")
	}
}
