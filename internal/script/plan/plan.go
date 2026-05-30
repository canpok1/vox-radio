package plan

import (
	"context"

	"github.com/canpok1/vox-radio/internal/model"
)

type Planner interface {
	Plan(ctx context.Context, summaries []model.Summary, show model.ShowConfig) (model.Rundown, error)
}
