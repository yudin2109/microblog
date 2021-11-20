package feed

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"netwitter/plain"
	"netwitter/schemas"
	"time"
)

type PersonalFeedItem struct {
	UserID    schemas.UserId `bson:"userId"`
	PostID    schemas.PostId `bson:"postId"`
	AuthorID  schemas.UserId `bson:"authorId"`
	Text      string         `bson:"text"`
	CreatedAt time.Time      `bson:"createdAt"`
}

type FeedStorage struct {
	feedCollection *mongo.Collection
}

func NewStorage(ctx context.Context, mongoUrl, dbName string) *FeedStorage {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUrl))
	if err != nil {
		panic(fmt.Sprintf("connect to mongo failed: %s", err))
	}

	feedCollection := mongoClient.Database(dbName).Collection("feed")
	err = ensureIndexes(ctx, feedCollection)
	if err != nil {
		panic(fmt.Sprintf("failed ensure index: %s", err))
	}

	return &FeedStorage{feedCollection: feedCollection}
}

func ensureIndexes(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"userId", 1}, {"createdAt", -1}},
	})
	if err != nil {
		return err
	}

	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"userId", 1}, {"postId", 1}, {"createdAt", -1}},
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *FeedStorage) PutPostToFeed(ctx context.Context, userId schemas.UserId, post schemas.Post) error {
	mongoQuery := bson.M{"userId": string(userId), "postId": post.ID}
	item := &PersonalFeedItem{
		UserID:    userId,
		PostID:    post.ID,
		AuthorID:  post.AuthorID,
		Text:      string(post.Content),
		CreatedAt: post.CreatedAt,
	}

	mongoOpts := options.Replace().SetUpsert(true)
	_, err := s.feedCollection.ReplaceOne(ctx, mongoQuery, item, mongoOpts)
	if err != nil {
		return err
	}
	return nil
}

func (s *FeedStorage) GetUserFeed(ctx context.Context, userId schemas.UserId, data plain.GetUserPostsPageData) ([]*schemas.Post, *plain.GetUserPostsPageData, error) {
	lastSeenPost, packSize, err := plain.CorrectDestruct(data)
	if err != nil {
		return nil, nil, err
	}

	mongoFilter := bson.M{"userId": string(userId)}
	if lastSeenPost != nil {
		mongoFilter["postId"] = bson.M{"$lte": *lastSeenPost}
	}

	searchPackSize := int64(packSize + 2)
	mongoOptions := &options.FindOptions{
		Limit: &searchPackSize,
		Sort:  bson.M{"createdAt": -1},
	}

	cursor, err := s.feedCollection.Find(ctx, mongoFilter, mongoOptions)
	if err != nil {
		return nil, nil, err
	}

	var allUserFeedItems []*PersonalFeedItem
	err = cursor.All(ctx, &allUserFeedItems)
	if err != nil {
		return nil, nil, err
	}

	if lastSeenPost != nil && len(allUserFeedItems) == 0 {
		return nil, nil, fmt.Errorf("invalid page token:%s", *lastSeenPost)
	}

	if lastSeenPost != nil && allUserFeedItems[0].PostID != *lastSeenPost {
		return nil, nil, fmt.Errorf("page not found:%s", *lastSeenPost)
	}

	if lastSeenPost != nil {
		allUserFeedItems = allUserFeedItems[1:]
	}

	var nextPageToken *plain.GetUserPostsPageData
	if len(allUserFeedItems) > packSize {
		nextPageToken = &plain.GetUserPostsPageData{
			LastSeenID: allUserFeedItems[packSize-1].PostID.ToBase64URL(),
			Size:       packSize,
		}
		allUserFeedItems = allUserFeedItems[:packSize]
	}

	feedPosts := make([]*schemas.Post, 0, len(allUserFeedItems))
	for i := range allUserFeedItems {
		feedPosts = append(feedPosts, &schemas.Post{
			ID:             allUserFeedItems[i].PostID,
			AuthorID:       allUserFeedItems[i].AuthorID,
			Content:        schemas.Text(allUserFeedItems[i].Text),
			CreatedAt:      allUserFeedItems[i].CreatedAt,
			LastModifiedAt: time.Now(),
			Version:        0,
		})
	}

	return feedPosts, nextPageToken, nil
}
