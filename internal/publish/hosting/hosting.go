package hosting

import (
	"context"
	"io"

	"github.com/canpok1/vox-radio/internal/model"
)

type Hosting interface {
	PutAudio(ctx context.Context, name string, r io.Reader) (url string, err error)
	PutFeed(ctx context.Context, feedXML []byte) (url string, err error)
	LoadEpisodes(ctx context.Context) (model.Episodes, error)
	SaveEpisodes(ctx context.Context, e model.Episodes) error
	DeleteAudio(ctx context.Context, name string) error
}

// Pusher is implemented by Hosting backends that require a post-publish push step.
type Pusher interface {
	Push(ctx context.Context) error
}
