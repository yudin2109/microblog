package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"netwitter/plain"
	"netwitter/schemas"
	"netwitter/storage"
	"strconv"
	"sync"
)

func NewHTTPHandler(storage storage.Storage) *HTTPHandler {
	return &HTTPHandler{
		Storage: storage,
	}
}

type HTTPHandler struct {
	StorageMu sync.RWMutex
	Storage   storage.Storage
}

type PutRequestData struct {
	Url string `json:"url"`
}

type PutResponseData struct {
	Key string `json:"key"`
}

type CreatePostRequestData struct {
	Text string `json:"text"`
}

type EditPostRequestData struct {
	Text string `json:"text"`
}

type GetUserPostsResponse struct {
	Posts    []schemas.PostData `json:"posts"`
	NextPage *string            `json:"nextPage,omitempty"`
}

func (h *HTTPHandler) HandleCreatePost(rw http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("System-Design-User-Id")
	if userId == "" {
		http.Error(rw, "no auth", http.StatusUnauthorized)
		return
	}

	var data CreatePostRequestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(rw, "bad body", http.StatusBadRequest)
		return
	}

	text := data.Text
	if text == "" {
		http.Error(rw, "text must not be empty", http.StatusBadRequest)
		return
	}

	newPost, err := h.Storage.PutPost(r.Context(), schemas.UserId(userId), schemas.Text(text))
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	postData := newPost.ToPostData()
	rawResponse, _ := json.Marshal(postData)
	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandleGetPost(rw http.ResponseWriter, r *http.Request) {
	postId := mux.Vars(r)["postId"]
	if postId == "" {
		http.Error(rw, "incorrect post id", http.StatusBadRequest)
		return
	}

	postIdBase64, err := schemas.IDFromRawString(postId)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	post, err := h.Storage.GetPost(r.Context(), postIdBase64)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	postData := post.ToPostData()
	rawResponse, _ := json.Marshal(postData)
	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandleGetUserPosts(rw http.ResponseWriter, r *http.Request) {
	userId := schemas.UserId(mux.Vars(r)["userId"])
	if userId == "" {
		http.Error(rw, "blank userId", http.StatusBadRequest)
		return
	}

	queryParams := r.URL.Query()
	var parsedPageData plain.GetUserPostsPageData
	if pageToken := queryParams.Get("page"); pageToken != "" {
		parsedPageData = plain.GetUserPostsPageData{LastSeenID: pageToken}
	}
	if rawSize := queryParams.Get("size"); rawSize != "" {
		parsedSize, err := strconv.ParseInt(rawSize, 10, 32)
		if err != nil {
			http.Error(rw, fmt.Sprintf("invalid page size: %s", err.Error()), http.StatusBadRequest)
			return
		}
		parsedPageData.Size = int(parsedSize)
	}
	if parsedPageData.Size == 0 {
		parsedPageData.Size = plain.DefaultPageSize
	}

	postList, nextPageToken, err := h.Storage.GetUserPosts(r.Context(), schemas.UserId(userId), parsedPageData)
	if err != nil {
		http.Error(rw, fmt.Sprintf("failed find posts:%s", err.Error()), http.StatusBadRequest)
		return
	}

	response := GetUserPostsResponse{
		Posts: make([]schemas.PostData, len(postList)),
	}

	for i, post := range postList {
		response.Posts[i] = post.ToPostData()
	}

	if nextPageToken != nil {
		nextPageEncoded := nextPageToken.LastSeenID
		response.NextPage = &nextPageEncoded
	}

	rawResponse, _ := json.Marshal(response)
	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandleEditPost(rw http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("System-Design-User-Id")
	if userId == "" {
		http.Error(rw, "no auth", http.StatusUnauthorized)
		return
	}

	postId := mux.Vars(r)["postId"]
	if postId == "" {
		http.Error(rw, "incorrect post id", http.StatusBadRequest)
		return
	}

	postIdBase64, err := schemas.IDFromRawString(postId)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var data EditPostRequestData
	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(rw, "bad body", http.StatusBadRequest)
		return
	}

	text := data.Text
	if text == "" {
		http.Error(rw, "text must not be empty", http.StatusBadRequest)
		return
	}

	post, err := h.Storage.GetPost(r.Context(), postIdBase64)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	if string(post.AuthorID) != userId {
		http.Error(rw, "you shall not pass", http.StatusForbidden)
		return
	}

	editedPost, err := h.Storage.EditPost(r.Context(), post.ID, post.AuthorID, schemas.Text(text))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	postData := editedPost.ToPostData()
	rawResponse, _ := json.Marshal(postData)
	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandlePing(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusOK)
}
