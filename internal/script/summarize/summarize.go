package summarize

import (
	"context"

	"github.com/canpok1/vox-radio/internal/model"
)

type Summarizer interface {
	Summarize(ctx context.Context, a model.Article) (model.Summary, error)
}
