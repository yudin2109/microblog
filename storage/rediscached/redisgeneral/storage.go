package redisgeneral

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"reflect"
	"time"
)

// Redis storage
// For consistence storable objects must have version (implement WithVersion interface)
// In redis objects stored in hashmap {"value" : data, "vers" : version}

type WithVersion interface {
	GetVersion() int
}

func NewStorage(client *redis.Client, valueType reflect.Type, ttl time.Duration) *Storage {
	if valueType == nil {
		panic("nil value type")
	}
	return &Storage{
		client:    client,
		valueType: valueType,
		ttl:       ttl,
	}
}

type Storage struct {
	client    *redis.Client
	valueType reflect.Type
	ttl       time.Duration
}

func (s *Storage) Get(ctx context.Context, key string) (WithVersion, bool, error) {
	marshalledResult, err := s.client.HGet(ctx, key, "value").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("redis error:%s", err.Error())
	}

	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}

	result, err := s.unmarshalJSON([]byte(marshalledResult))
	if err != nil {
		return nil, false, fmt.Errorf("incorrect json:%s", err.Error())
	}
	return result, true, nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis failed delete: %s", err.Error())
	}
	return nil
}

func (s *Storage) unmarshalJSON(valueJSON []byte) (WithVersion, error) {
	unmarshalled := reflect.New(s.valueType)
	err := json.Unmarshal(valueJSON, unmarshalled.Interface())
	if err != nil {
		return nil, fmt.Errorf("unmarshal json failed: %s", err.Error())
	}
	return reflect.Indirect(unmarshalled).Interface().(WithVersion), nil
}

//go:embed set_fresh.lua
var setWithFreshnessSource string
var setWithFreshnessScript = redis.NewScript(setWithFreshnessSource)

func (s *Storage) SetWithFreshness(ctx context.Context, key string, value WithVersion) (WithVersion, error) {
	marshalled, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshalling failed: %s", err.Error())
	}

	keys := []string{key}
	argv := []interface{}{marshalled, value.GetVersion(), s.ttl.Milliseconds()}
	returned, err := setWithFreshnessScript.Run(ctx, s.client, keys, argv...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis error: %s", err.Error())
	}
	return s.unmarshalJSON([]byte(returned.(string)))

}
