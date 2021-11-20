package main

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"netwitter/handlers"
	"netwitter/storage"
	"netwitter/storage/inmemory"
	"netwitter/storage/mongostorage"
	"netwitter/storage/rediscached"
	"os"
	"strconv"
	"time"
)

const defaultServerPort = 8080

func NewServer() *http.Server {
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = strconv.Itoa(defaultServerPort)
	}

	postStorage := GetPostStorage()
	handler := handlers.NewHTTPHandler(postStorage)

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleGetPost).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleEditPost).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/users/{userId}/posts", handler.HandleGetUserPosts).Methods(http.MethodGet)
	r.HandleFunc("/maintenance/ping", handler.HandlePing).Methods(http.MethodGet)

	return &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%s", serverPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

func GetPostStorage() storage.Storage {
	switch mode := os.Getenv("STORAGE_MODE"); mode {
	case "inmemory":
		return inmemory.NewInMemoryStorage()
	case "mongo":
		return ConstructMongoStorage()
	case "cached":
		return ConstructCachedStorage()
	default:
		panic(fmt.Errorf("unexpected storage mode: %q", mode))
	}
}

func ConstructMongoStorage() storage.Storage {
	dbUrl := os.Getenv("MONGO_URL")
	if dbUrl == "" {
		panic(errors.New("empty mongo url"))
	}
	dbName := os.Getenv("MONGO_DBNAME")
	if dbName == "" {
		panic(errors.New("empty mongo name"))
	}
	return mongostorage.NewStorage(dbUrl, dbName)
}

func ConstructCachedStorage() storage.Storage {
	mongoStorage := ConstructMongoStorage()

	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		panic(errors.New("empty redis url"))
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisUrl})

	return rediscached.NewCachedStorage(mongoStorage, redisClient, 5*time.Minute)
}

func main() {
	srv := NewServer()
	log.Printf("Start serving on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
