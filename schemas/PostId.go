package schemas

import (
	"encoding/base64"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PostId primitive.ObjectID

const LEN = 12

func (id PostId) ToBase64URL() string {
	bytes := [LEN]byte(id)
	return base64.URLEncoding.EncodeToString(bytes[:])
}

func IDFromRawString(s string) (PostId, error) {
	bytes, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return PostId{}, err
	}

	if len(bytes) != LEN {
		return PostId{}, fmt.Errorf("incorrect length of postid, got %d", len(bytes))
	}
	var array [LEN]byte
	copy(array[:], bytes)
	return array, nil
}

func (id PostId) Hex() string {
	return primitive.ObjectID(id).Hex()
}
