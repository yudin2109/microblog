package users

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"netwitter/schemas"
)

type SubscriptionInfo struct {
	SubscriberID schemas.UserId `bson:"subscriberId"`
	TargetUserID schemas.UserId `bson:"targetUserId"`
}

type UsersStorage struct {
	usersCollection *mongo.Collection
}

func NewStorage(ctx context.Context, mongoUrl, dbName string) *UsersStorage {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUrl))
	if err != nil {
		panic(fmt.Sprintf("connect to mongo failed: %s", err))
	}

	usersCollestion := mongoClient.Database(dbName).Collection("subscriptions")
	err = ensureIndexes(ctx, usersCollestion)
	if err != nil {
		panic(fmt.Sprintf("failed ensure index: %s", err))
	}

	return &UsersStorage{usersCollection: usersCollestion}
}

func ensureIndexes(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"subscriberId", 1}, {"targetUserId", 1}},
	})
	if err != nil {
		return err
	}

	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"targetUserId", 1}},
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *UsersStorage) MakeSubscription(ctx context.Context, subscriber schemas.UserId, to schemas.UserId) error {
	mongoQuery := bson.M{"subscriberId": string(subscriber), "targetUserId": string(to)}
	mongoOpts := options.Replace().SetUpsert(true)
	subscrInfo := &SubscriptionInfo{
		SubscriberID: subscriber,
		TargetUserID: to,
	}
	_, err := s.usersCollection.ReplaceOne(ctx, mongoQuery, subscrInfo, mongoOpts)
	if err != nil {
		return fmt.Errorf("subscription insertion failed: %s", err.Error())
	}

	return nil
}

func (s *UsersStorage) GetUserSubscriptions(ctx context.Context, userId schemas.UserId) ([]schemas.UserId, error) {
	mongoQuery := bson.M{"subscriberId": string(userId)}
	cursor, err := s.usersCollection.Find(ctx, mongoQuery)
	if err != nil {
		return nil, fmt.Errorf("mongo search failed: %s", err.Error())
	}

	var allUserSubscriptions []*SubscriptionInfo
	err = cursor.All(ctx, &allUserSubscriptions)
	if err != nil {
		return nil, fmt.Errorf("putting feed from mongo failed: %s", err.Error())
	}

	userSubList := make([]schemas.UserId, 0, len(allUserSubscriptions))
	for i := range allUserSubscriptions {
		userSubList = append(userSubList, allUserSubscriptions[i].TargetUserID)
	}
	return userSubList, nil
}

// copy-paste of previous)
func (s *UsersStorage) GetUserSubscribers(ctx context.Context, userId schemas.UserId) ([]schemas.UserId, error) {
	mongoQuery := bson.M{"targetUserId": string(userId)}
	cursor, err := s.usersCollection.Find(ctx, mongoQuery)
	if err != nil {
		return nil, fmt.Errorf("mongo search failed: %s", err.Error())
	}

	var allUserSubscribers []*SubscriptionInfo
	err = cursor.All(ctx, &allUserSubscribers)
	if err != nil {
		return nil, fmt.Errorf("putting feed from mongo failed: %s", err.Error())
	}

	userSubList := make([]schemas.UserId, 0, len(allUserSubscribers))
	for i := range allUserSubscribers {
		userSubList = append(userSubList, allUserSubscribers[i].SubscriberID)
	}
	return userSubList, nil
}
