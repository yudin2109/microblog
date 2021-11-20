package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"netwitter/handlers"
	"netwitter/storage"
	"netwitter/storage/inmemory"
	"netwitter/storage/mongostorage"
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
	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods("POST")
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleGetPost).Methods("GET")
	r.HandleFunc("/api/v1/users/{userId}/posts", handler.HandleGetUserPosts).Methods("GET")
	r.HandleFunc("/maintenance/ping", handler.HandlePing).Methods("GET")

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

func main() {
	srv := NewServer()
	log.Printf("Start serving on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
