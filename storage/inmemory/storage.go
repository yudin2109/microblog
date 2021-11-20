package inmemory

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"netwitter/plain"
	"netwitter/schemas"
	"sort"
	"sync"
	"time"
)

type MemoryStorage struct {
	mu sync.RWMutex

	postById     map[schemas.PostId]*schemas.Post
	postByAuthor map[schemas.UserId][]*schemas.Post
}

func NewInMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		postById:     map[schemas.PostId]*schemas.Post{},
		postByAuthor: map[schemas.UserId][]*schemas.Post{},
	}
}
func (s *MemoryStorage) PutPost(_ context.Context, userId schemas.UserId, text schemas.Text) (*schemas.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newPost := &schemas.Post{
		ID:        schemas.PostId(primitive.NewObjectID()),
		AuthorID:  userId,
		Content:   text,
		CreatedAt: time.Now(),
	}

	s.postById[newPost.ID] = newPost
	userPostList := s.postByAuthor[userId]
	userPostList = append(userPostList, newPost)
	for i := len(userPostList) - 1; i > 0 && userPostList[i].ID.Hex() < userPostList[i-1].ID.Hex(); i-- {
		userPostList[i-1], userPostList[i] = userPostList[i], userPostList[i-1]
	}
	s.postByAuthor[userId] = userPostList

	var result schemas.Post
	result = *newPost
	return &result, nil
}

func (s *MemoryStorage) GetPost(_ context.Context, postId schemas.PostId) (*schemas.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	post, ok := s.postById[postId]
	if !ok {
		return nil, fmt.Errorf("not found: %s", postId)
	}

	var result schemas.Post
	result = *post
	return &result, nil
}

func (s *MemoryStorage) GetUserPosts(_ context.Context, authorID schemas.UserId, pageData plain.GetUserPostsPageData) ([]*schemas.Post, *plain.GetUserPostsPageData, error) {
	lastSeenID, size, err := plain.CorrectDestruct(pageData)
	if err != nil {
		return nil, nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	userPostList := s.postByAuthor[authorID]

	lastSeenIndex := len(userPostList)
	if lastSeenID != nil {
		lastSeenIndex = sort.Search(len(userPostList), func(i int) bool {
			return userPostList[i].ID.Hex() >= lastSeenID.Hex()
		})
		if lastSeenIndex == len(userPostList) || userPostList[lastSeenIndex].ID != *lastSeenID {
			return nil, nil, fmt.Errorf("incorrect page token: %s", lastSeenID)
		}
	}

	nextPackStart := MaxInt(0, lastSeenIndex-1)
	nextPackEnd := MaxInt(0, nextPackStart+1-size)
	packSize := lastSeenIndex - nextPackEnd

	pack := make([]*schemas.Post, packSize)
	for i := 0; i < packSize; i++ {
		pack[i] = userPostList[nextPackStart-i].Copy()
	}

	var nextPageToken *plain.GetUserPostsPageData
	if nextPackEnd > 0 {
		nextPageToken = &plain.GetUserPostsPageData{
			LastSeenID: userPostList[nextPackEnd].ID.ToBase64URL(),
			Size:       size,
		}
	}
	return pack, nextPageToken, nil
}

func MaxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
