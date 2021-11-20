package feed

import (
	"context"
	"netwitter/schemas"
	"netwitter/storage"
)

type FeedManager struct {
	postStorage storage.Storage
	userStorage storage.UsersStorage
	feedStorage storage.FeedStorage
}

func NewFeedManager(postStorage storage.Storage, userStorage storage.UsersStorage, feedStorage storage.FeedStorage) *FeedManager {
	return &FeedManager{
		postStorage: postStorage,
		userStorage: userStorage,
		feedStorage: feedStorage,
	}
}

func (fm *FeedManager) SpreadPostOverSubscribers(ctx context.Context, userID schemas.UserId, postID schemas.PostId) error {
	subscribers, err := fm.userStorage.GetUserSubscribers(ctx, userID)
	if err != nil {
		return err
	}

	post, err := fm.postStorage.GetPost(ctx, postID)
	if err != nil {
		return err
	}

	for _, subscriber := range subscribers {
		err = fm.feedStorage.PutPostToFeed(ctx, subscriber, *post)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fm *FeedManager) CollectPostsToPersonalFeed(ctx context.Context, subscriber schemas.UserId, from schemas.UserId) error {
	postsIterator, err := fm.postStorage.GetAllPostsFromUser(ctx, from)
	if err != nil {
		return err
	}

	for p := postsIterator.GetNextPost(ctx); p != nil; p = postsIterator.GetNextPost(ctx) {
		err = fm.feedStorage.PutPostToFeed(ctx, subscriber, *p)
		if err != nil {
			return err
		}
	}
	return nil
}
