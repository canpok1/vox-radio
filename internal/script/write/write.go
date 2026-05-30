package write

import (
	"context"

	"github.com/canpok1/vox-radio/internal/model"
)

type Writer interface {
	Write(ctx context.Context, corner model.Corner, summary model.Summary, show model.ShowConfig) ([]model.Line, error)
}
