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
	EditPost(ctx context.Context, postId schemas.PostId, authorId schemas.UserId, text schemas.Text) (*schemas.Post, error)
	GetUserPosts(ctx context.Context, authorID schemas.UserId, pageData plain.GetUserPostsPageData) (_ []*schemas.Post, nextPage *plain.GetUserPostsPageData, _ error)
	GetAllPostsFromUser(ctx context.Context, authorId schemas.UserId) (plain.PostsIterator, error)
}

type UsersStorage interface {
	MakeSubscription(ctx context.Context, subscriber schemas.UserId, to schemas.UserId) error
	GetUserSubscriptions(ctx context.Context, userId schemas.UserId) ([]schemas.UserId, error)
	GetUserSubscribers(ctx context.Context, userId schemas.UserId) ([]schemas.UserId, error)
}

type FeedStorage interface {
	PutPostToFeed(ctx context.Context, userId schemas.UserId, post schemas.Post) error
	GetUserFeed(ctx context.Context, userId schemas.UserId, data plain.GetUserPostsPageData) ([]*schemas.Post, *plain.GetUserPostsPageData, error)
}
