package model_test

import (
	"testing"

	"github.com/canpok1/vox-radio/internal/model"
)

func TestText(t *testing.T) {
	type texter interface{ Text() string }
	cases := []struct {
		name string
		obj  texter
		want string
	}{
		{
			name: "Article/body_preferred",
			obj:  model.Article{DedupKey: "k", Title: "t", Description: "feed", Body: "html"},
			want: "html",
		},
		{
			name: "Article/description_fallback",
			obj:  model.Article{DedupKey: "k", Title: "t", Description: "feed"},
			want: "feed",
		},
		{
			name: "Article/both_empty",
			obj:  model.Article{DedupKey: "k", Title: "t"},
			want: "",
		},
		{
			name: "RundownArticle/body_preferred",
			obj:  model.RundownArticle{DedupKey: "k", Title: "t", Description: "feed", Body: "html", Points: []string{}},
			want: "html",
		},
		{
			name: "RundownArticle/description_fallback",
			obj:  model.RundownArticle{DedupKey: "k", Title: "t", Description: "feed", Points: []string{}},
			want: "feed",
		},
		{
			name: "RundownArticle/both_empty",
			obj:  model.RundownArticle{DedupKey: "k", Title: "t", Points: []string{}},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.obj.Text(); got != tc.want {
				t.Errorf("Text() = %q, want %q", got, tc.want)
			}
		})
	}
}
