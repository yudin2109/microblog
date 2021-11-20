package plain

import (
	"context"
	"fmt"
	"netwitter/schemas"
)

const DefaultPageSize = 10

type GetUserPostsPageData struct {
	LastSeenID string
	Size       int
}

func CorrectDestruct(pageData GetUserPostsPageData) (*schemas.PostId, int, error) {
	size := pageData.Size
	switch {
	case size < 0:
		return nil, 0, fmt.Errorf("page size must not be negative: %d", size)
	case size == 0:
		size = DefaultPageSize
	}

	var firstPostID *schemas.PostId
	if rawID := pageData.LastSeenID; rawID != "" {
		postID, err := schemas.IDFromRawString(pageData.LastSeenID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid post id: %s", err.Error())
		}
		firstPostID = &postID
	}

	return firstPostID, size, nil
}

type PostsIterator interface {
	GetNextPost(ctx context.Context) *schemas.Post
}
