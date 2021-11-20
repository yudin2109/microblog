package schemas

import (
	"time"
)

type UserId string
type Text string

type Post struct {
	ID             PostId    `bson:"_id"`
	Version        int       `bson:"version"`
	AuthorID       UserId    `bson:"authorId"`
	Content        Text      `bson:"text"`
	CreatedAt      time.Time `bson:"createdAt"`
	LastModifiedAt time.Time `bson:"lastModifiedAt"`
}

type PostData struct {
	ID        string `json:"id"`
	Content   Text   `json:"text"`
	AuthorID  string `json:"authorId"`
	CreatedAt string `json:"createdAt"`
}

func (p *Post) ToPostData() PostData {
	return PostData{
		ID:        p.ID.ToBase64URL(),
		Content:   p.Content,
		AuthorID:  string(p.AuthorID),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (p Post) Copy() *Post {
	return &p
}
