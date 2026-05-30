package direct

import (
	"context"

	"github.com/canpok1/vox-radio/internal/model"
)

type Director interface {
	Direct(ctx context.Context, lines []model.Line, se model.SECatalog) (model.Script, error)
}
