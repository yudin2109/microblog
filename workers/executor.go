package workers

import (
	"context"
	"netwitter/feed"
	"netwitter/schemas"
)

type PostsTasksExecutor struct {
	feedManager feed.FeedManager
}

func NewPostsTasksExecutor(feedManager feed.FeedManager) *PostsTasksExecutor {
	return &PostsTasksExecutor{feedManager: feedManager}
}

func (pte *PostsTasksExecutor) ExecuteSpreadPostOverSubscribers(userId string, postId string) error {
	ctx := context.Background()
	userIdInSchemas := schemas.UserId(userId)
	postIdInSchemas, err := schemas.IDFromText(postId)
	if err != nil {
		panic(err)
	}
	err = pte.feedManager.SpreadPostOverSubscribers(ctx, userIdInSchemas, postIdInSchemas)
	if err != nil {
		return err
	}
	return nil
}

func (pte *PostsTasksExecutor) ExecuteCollectPostsToPersonalFeed(subscriber string, from string) error {
	ctx := context.Background()
	subscriberInSchemas := schemas.UserId(subscriber)
	sourceInSchemas := schemas.UserId(from)

	err := pte.feedManager.CollectPostsToPersonalFeed(ctx, subscriberInSchemas, sourceInSchemas)
	if err != nil {
		return err
	}
	return nil
}

func (pte *PostsTasksExecutor) GetCommandsMapping() map[string]interface{} {
	return map[string]interface{}{
		"SpreadPostOverSubscribers":  pte.ExecuteSpreadPostOverSubscribers,
		"CollectPostsToPersonalFeed": pte.ExecuteCollectPostsToPersonalFeed,
	}
}
