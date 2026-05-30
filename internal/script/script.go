package script

import (
	"context"

	"github.com/canpok1/vox-radio/internal/model"
)

type ScriptGenerator interface {
	Generate(ctx context.Context, articles []model.Article, show model.ShowConfig) (model.Script, error)
}
