package users

import (
	"context"
	"netwitter/plain"
	"netwitter/schemas"
	"netwitter/storage"
	"netwitter/workers"
)

type UsersManager struct {
	usersStorage storage.UsersStorage
	feedStorage  storage.FeedStorage
	scheduler    workers.Scheduler
}

func NewUsersManager(usersStorage storage.UsersStorage, feedStorage storage.FeedStorage, scheduler workers.Scheduler) *UsersManager {
	return &UsersManager{usersStorage: usersStorage, feedStorage: feedStorage, scheduler: scheduler}
}

func (um *UsersManager) MakeSubscription(ctx context.Context, subscriber schemas.UserId, to schemas.UserId) error {
	err := um.usersStorage.MakeSubscription(ctx, subscriber, to)
	if err != nil {
		return err
	}

	err = um.scheduler.PublishCollectPostsToPersonalFeed(subscriber, to)
	if err != nil {
		return err
	}
	return nil
}

func (um *UsersManager) GetUserSubscriptions(ctx context.Context, userId schemas.UserId) ([]schemas.UserId, error) {
	return um.usersStorage.GetUserSubscriptions(ctx, userId)
}

func (um *UsersManager) GetUserSubscribers(ctx context.Context, userId schemas.UserId) ([]schemas.UserId, error) {
	return um.usersStorage.GetUserSubscribers(ctx, userId)
}
func (um *UsersManager) GetUserFeed(ctx context.Context, userId schemas.UserId, page plain.GetUserPostsPageData) ([]*schemas.Post, *plain.GetUserPostsPageData, error) {
	return um.feedStorage.GetUserFeed(ctx, userId, page)
}
