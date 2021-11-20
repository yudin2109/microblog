package mongostorage

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"netwitter/plain"
	"netwitter/schemas"
	"netwitter/workers"
	"time"
)

const collName = "posts"

type storage struct {
	postsCollection *mongo.Collection
	scheduler       workers.Scheduler
}

func NewStorage(mongoURL string, mongoName string, scheduler workers.Scheduler) *storage {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		panic(err)
	}

	postsCollection := client.Database(mongoName).Collection(collName)

	ensureIndexes(ctx, postsCollection)

	return &storage{
		postsCollection: postsCollection,
		scheduler:       scheduler,
	}
}

func ensureIndexes(ctx context.Context, collection *mongo.Collection) {
	indexModels := mongo.IndexModel{
		Keys: bson.D{{"authorId", 1}, {"_id", -1}},
	}
	_, err := collection.Indexes().CreateOne(ctx, indexModels)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes %w", err))
	}
}

func (s *storage) PutPost(ctx context.Context, userId schemas.UserId, text schemas.Text) (*schemas.Post, error) {
	newPost := &schemas.Post{
		ID:             schemas.PostId(primitive.NewObjectID()),
		AuthorID:       userId,
		Content:        text,
		CreatedAt:      s.Now(),
		LastModifiedAt: s.Now(),
	}

	_, err := s.postsCollection.InsertOne(ctx, newPost)
	if err != nil {
		return nil, fmt.Errorf("insertion failed: %s", err.Error())
	}
	err = s.scheduler.PublishSpreadPostOverSubs(userId, newPost.ID)
	if err != nil {
		return nil, err
	}
	return newPost, nil
}

func (s *storage) GetPost(ctx context.Context, postId schemas.PostId) (*schemas.Post, error) {
	var post schemas.Post
	err := s.postsCollection.FindOne(ctx, bson.M{"_id": postId}).Decode(&post)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("not found post %s, cause %s", postId, err.Error())
		}
		return nil, fmt.Errorf("failed to extract, cause %s", err.Error())
	}
	return &post, nil
}

func (s *storage) GetUserPosts(ctx context.Context, authorID schemas.UserId, pageData plain.GetUserPostsPageData) ([]*schemas.Post, *plain.GetUserPostsPageData, error) {
	lastSeenID, size, err := plain.CorrectDestruct(pageData)
	if err != nil {
		return nil, nil, err
	}

	mongoFilter := bson.M{"authorId": string(authorID)}
	if lastSeenID != nil {
		mongoFilter["_id"] = bson.M{"$lte": *lastSeenID}
	}
	optionsLimit := int64(size + 2) // with redundant previous and next
	filterOptions := &options.FindOptions{
		Limit: &optionsLimit,
		Sort:  bson.M{"_id": -1},
	}
	cursor, err := s.postsCollection.Find(ctx, mongoFilter, filterOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("search failed: %s", err.Error())
	}
	var postList []*schemas.Post
	if err = cursor.All(ctx, &postList); err != nil {
		return nil, nil, fmt.Errorf("posts mapping failed: %s", err.Error())
	}

	if lastSeenID != nil && len(postList) == 0 {
		return nil, nil, fmt.Errorf("something went wrong(no posts but lastseen not null)")
	}

	if lastSeenID != nil && postList[0].ID != *lastSeenID {
		return nil, nil, fmt.Errorf("something went wrong(lastseen missmatch): %s", *lastSeenID)
	}

	if lastSeenID != nil {
		postList = postList[1:]
	}

	var nextPage *plain.GetUserPostsPageData
	if len(postList) > size {
		//page is overfilled, there is next element [...]+
		nextPage = &plain.GetUserPostsPageData{
			LastSeenID: postList[size-1].ID.ToBase64URL(),
			Size:       size,
		}
		postList = postList[:size]
	}

	return postList, nextPage, nil
}

func (s *storage) EditPost(ctx context.Context, postId schemas.PostId, authorId schemas.UserId, text schemas.Text) (*schemas.Post, error) {
	mongoSelector := bson.D{{"_id", postId}}
	mongoCommand := bson.D{
		{
			"$set", bson.D{
				{"text", text},
				{"lastModifiedAt", s.Now()},
			},
		},
		{
			"$inc", bson.D{{"version", 1}},
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	result := s.postsCollection.FindOneAndUpdate(ctx, mongoSelector, mongoCommand, opts)

	var editedPost schemas.Post
	err := result.Decode(&editedPost)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("not found:%s", postId)
		}
		return nil, fmt.Errorf("mongo error:%s", err.Error())
	}
	err = s.scheduler.PublishSpreadPostOverSubs(editedPost.AuthorID, editedPost.ID)
	if err != nil {
		return nil, err
	}
	return &editedPost, nil
}

type MongoPostsIterator struct {
	cursor *mongo.Cursor
}

func (mpi *MongoPostsIterator) GetNextPost(ctx context.Context) *schemas.Post {
	hasNext := mpi.cursor.Next(ctx)
	if !hasNext {
		return nil
	}

	var post schemas.Post
	err := mpi.cursor.Decode(&post)
	if err != nil {
		return nil
	}
	return &post
}

func (s *storage) GetAllPostsFromUser(ctx context.Context, authorId schemas.UserId) (plain.PostsIterator, error) {
	mongoFilter := bson.M{"authorId": string(authorId)}

	cursor, err := s.postsCollection.Find(ctx, mongoFilter)
	if err != nil {
		return nil, err
	}

	return &MongoPostsIterator{cursor: cursor}, nil
}

func (s *storage) Now() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}
