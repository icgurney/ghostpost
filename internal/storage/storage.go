package storage

import (
	"context"
	"io"
)

type Storage interface {
	SaveEmail(ctx context.Context, id string, body io.Reader) error
}
