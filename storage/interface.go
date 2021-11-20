package storage

import (
	"context"
	"errors"
	"fmt"
	"netwitter/plain"
	"netwitter/schemas"
)

var (
	StorageError = errors.New("storage")
	ErrCollision = fmt.Errorf("%w.collision", StorageError)
	ErrNotFound  = fmt.Errorf("%w.not_found", StorageError)
)

type Storage interface {
	PutPost(ctx context.Context, userId schemas.UserId, text schemas.Text) (*schemas.Post, error)
	GetPost(ctx context.Context, postId schemas.PostId) (*schemas.Post, error)
	GetUserPosts(ctx context.Context, authorID schemas.UserId, pageData plain.GetUserPostsPageData) (_ []*schemas.Post, nextPage *plain.GetUserPostsPageData, _ error)
}
