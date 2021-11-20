package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"netwitter/feed"
	"netwitter/handlers"
	"netwitter/storage/mongostorage"
	"netwitter/users"
	"netwitter/workers"
	"os"
	"time"
)

const defaultServerPort = "8080"

func Start() error {
	switch mode := os.Getenv("APP_MODE"); mode {
	case "SERVER":
		return runAsServer()
	case "WORKER":
		return runAsWorker()
	default:
		panic(fmt.Errorf("unexpected app mode: %s", mode))
	}
}

func runAsServer() error {
	ctx := context.Background()

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = defaultServerPort
	}

	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		panic(fmt.Errorf("empty mongo url"))
	}
	dbName := os.Getenv("MONGO_DBNAME")
	if dbName == "" {
		panic(fmt.Errorf("empty mongo dbname"))
	}

	brokerURL := os.Getenv("REDIS_URL")
	if brokerURL == "" {
		panic(fmt.Errorf("nempty broker url"))
	}

	usersStorage := users.NewStorage(ctx, mongoURL, dbName)
	feedStorage := feed.NewStorage(ctx, mongoURL, dbName)

	scheduler := workers.NewScheduler(brokerURL)
	postsStorage := mongostorage.NewStorage(mongoURL, dbName, *scheduler)
	feedManager := feed.NewFeedManager(postsStorage, usersStorage, feedStorage)
	usersManager := users.NewUsersManager(usersStorage, feedStorage, *scheduler)

	executor := workers.NewPostsTasksExecutor(*feedManager)

	err := scheduler.Register(*executor)
	if err != nil {
		panic(err)
	}

	handler := handlers.NewHTTPHandler(postsStorage, *usersManager)
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleGetPost).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleEditPost).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/users/{userId}/posts", handler.HandleGetUserPosts).Methods(http.MethodGet)

	r.HandleFunc("/api/v1/subscriptions", handler.HandleGetUserSubscriptions).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/subscribers", handler.HandleGetUserSubscribers).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/users/{userId}/subscribe", handler.HandleSubscribeUser).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/feed", handler.HandleGetUserFeed).Methods(http.MethodGet)

	r.HandleFunc("/maintenance/ping", handler.HandlePing).Methods(http.MethodGet)

	server := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%s", serverPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Start serving at %s", server.Addr)
	return server.ListenAndServe()
}

func runAsWorker() error {
	ctx := context.Background()

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = defaultServerPort
	}

	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		panic(fmt.Errorf("empty mongo url"))
	}
	dbName := os.Getenv("MONGO_DBNAME")
	if dbName == "" {
		panic(fmt.Errorf("empty mongo dbname"))
	}

	brokerURL := os.Getenv("REDIS_URL")
	if brokerURL == "" {
		panic(fmt.Errorf("nempty broker url"))
	}

	scheduler := workers.NewScheduler(brokerURL)

	usersStorage := users.NewStorage(ctx, mongoURL, dbName)
	feedStorage := feed.NewStorage(ctx, mongoURL, dbName)
	postsStorage := mongostorage.NewStorage(mongoURL, dbName, *scheduler)

	feedManager := feed.NewFeedManager(postsStorage, usersStorage, feedStorage)

	executor := workers.NewPostsTasksExecutor(*feedManager)
	err := scheduler.Register(*executor)
	if err != nil {
		panic(err)
	}

	return scheduler.Listen()
}

func main() {
	log.Println(Start())
}
