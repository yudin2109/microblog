package rediscached

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"netwitter/plain"
	"netwitter/schemas"
	"netwitter/storage"
	"netwitter/storage/rediscached/redisgeneral"
	"reflect"
	"time"
)

type CachedPostsPack struct {
	Posts        []*schemas.Post
	NextPageData *plain.GetUserPostsPageData
}

func (cpp CachedPostsPack) GetVersion() int {
	vers := 0
	for i := range cpp.Posts {
		vers += cpp.Posts[i].Version
	}
	return vers
}

type CachedStorage struct {
	persistentStorage   storage.Storage
	postCache           *redisgeneral.Storage
	firstPostsPackCache *redisgeneral.Storage
}

func NewCachedStorage(persistentStorage storage.Storage, client *redis.Client, cacheTTL time.Duration) *CachedStorage {
	postCache := redisgeneral.NewStorage(client, reflect.TypeOf(schemas.Post{}), cacheTTL)
	firstPostsPackCache := redisgeneral.NewStorage(client, reflect.TypeOf(CachedPostsPack{}), cacheTTL)
	return &CachedStorage{
		persistentStorage:   persistentStorage,
		postCache:           postCache,
		firstPostsPackCache: firstPostsPackCache,
	}
}

func (cs *CachedStorage) PutPost(ctx context.Context, userId schemas.UserId, text schemas.Text) (*schemas.Post, error) {
	post, err := cs.persistentStorage.PutPost(ctx, userId, text)
	if err != nil {
		return nil, err
	}
	_, err = cs.postCache.SetWithFreshness(ctx, cs.getKeyForPost(post.ID), post)
	if err != nil {
		return nil, err
	}

	err = cs.firstPostsPackCache.Delete(ctx, cs.getKeyForFPP(userId))
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (cs *CachedStorage) GetPost(ctx context.Context, postId schemas.PostId) (*schemas.Post, error) {
	postKey := cs.getKeyForPost(postId)

	cachedPost, isFound, err := cs.postCache.Get(ctx, postKey)
	if err != nil {
		return nil, err
	}
	if isFound {
		return cachedPost.(schemas.Post).Copy(), nil
	}

	actualPost, err := cs.persistentStorage.GetPost(ctx, postId)
	if err != nil {
		return nil, err
	}
	// Trying to cache actual data
	cachedPost, err = cs.postCache.SetWithFreshness(ctx, postKey, actualPost)
	if err != nil {
		return nil, err
	}
	return cachedPost.(schemas.Post).Copy(), nil
}

func (cs *CachedStorage) EditPost(ctx context.Context, postId schemas.PostId, authorId schemas.UserId, text schemas.Text) (*schemas.Post, error) {
	editedPost, err := cs.persistentStorage.EditPost(ctx, postId, authorId, text)
	if err != nil {
		return nil, err
	}
	_, err = cs.postCache.SetWithFreshness(ctx, cs.getKeyForPost(postId), editedPost)
	if err != nil {
		return nil, err
	}
	err = cs.firstPostsPackCache.Delete(ctx, cs.getKeyForFPP(authorId))
	if err != nil {
		return nil, err
	}
	return editedPost, nil
}

func (cs *CachedStorage) GetUserPosts(ctx context.Context, authorID schemas.UserId, pageData plain.GetUserPostsPageData) (_ []*schemas.Post, nextPage *plain.GetUserPostsPageData, _ error) {
	if pageData.LastSeenID != "" || pageData.Size < 0 || pageData.Size > plain.DefaultPageSize {
		return cs.persistentStorage.GetUserPosts(ctx, authorID, pageData)
	}
	if pageData.Size == 0 {
		pageData.Size = plain.DefaultPageSize
	}

	// first page of user posts is cached
	fppKey := cs.getKeyForFPP(authorID)
	rawCached, found, err := cs.firstPostsPackCache.Get(ctx, fppKey)

	if err != nil {
		return nil, nil, err
	}
	if found {
		// Hey bro, nice compiler
		var cppRef CachedPostsPack
		cppRef = rawCached.(CachedPostsPack)
		return cs.constructPageDataFromCachedData(&cppRef, pageData)
	}

	firstPage, nextPage, err := cs.persistentStorage.GetUserPosts(ctx, authorID, plain.GetUserPostsPageData{LastSeenID: pageData.LastSeenID})
	if err != nil {
		return nil, nil, err
	}

	rawCached, err = cs.firstPostsPackCache.SetWithFreshness(ctx, fppKey, &CachedPostsPack{firstPage, nextPage})
	if err != nil {
		return nil, nil, err
	}

	// Hey bro, nice compiler
	var cppRef CachedPostsPack
	cppRef = rawCached.(CachedPostsPack)
	return cs.constructPageDataFromCachedData(&cppRef, pageData)
}

func (cs *CachedStorage) getKeyForPost(postID schemas.PostId) string {
	return fmt.Sprintf("ntwt:posts:%s", postID.ToBase64URL())
}

func (cs *CachedStorage) getKeyForFPP(userId schemas.UserId) string {
	return fmt.Sprintf("ntwt:fppack:%s", userId)
}

func (cs *CachedStorage) constructPageDataFromCachedData(cachedPage *CachedPostsPack, page plain.GetUserPostsPageData) ([]*schemas.Post, *plain.GetUserPostsPageData, error) {
	switch {
	case len(cachedPage.Posts) == page.Size:
		return cachedPage.Posts, cachedPage.NextPageData, nil
	case len(cachedPage.Posts) < page.Size:
		return cachedPage.Posts, nil, nil
	case len(cachedPage.Posts) > page.Size:
		return cachedPage.Posts[:page.Size], &plain.GetUserPostsPageData{LastSeenID: cachedPage.Posts[page.Size-1].ID.ToBase64URL(), Size: page.Size}, nil
	default:
		panic("wtf")
	}
}
