package model_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestNewRundownArticle_DescriptionAndBody(t *testing.T) {
	a := model.NewRundownArticle("k", "url", "title", "desc", "body", []string{"p"}, "src", "auth", "pub")
	if a.Description != "desc" {
		t.Errorf("Description = %q, want %q", a.Description, "desc")
	}
	if a.Body != "body" {
		t.Errorf("Body = %q, want %q", a.Body, "body")
	}
}
