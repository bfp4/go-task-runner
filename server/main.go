package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

type postInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

const localPostIDStart = 101

var (
	posts   []Post
	postsMu sync.RWMutex
	nextID  = 101
)

func main() {
	r := gin.Default()

	r.GET("/all-posts", getAllPosts)
	r.GET("/posts/:userID", getPostsByUserID)

	authorized := r.Group("/", authMiddleware())
	authorized.POST("/create-post", createPost)
	authorized.PUT("/posts/:id", updatePost)
	authorized.DELETE("/posts/:id", deletePost)

	r.Run(":8080")
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-ID header"})
			return
		}

		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid X-User-ID header"})
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func createPost(c *gin.Context) {
	userID := c.MustGet("userID").(int)

	var input postInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post := Post{
		ID:     nextID,
		Title:  input.Title,
		Body:   input.Body,
		UserID: userID,
	}

	nextID++
	postsMu.Lock()
	posts = append(posts, post)
	postsMu.Unlock()
	c.JSON(http.StatusCreated, post)
}

func updatePost(c *gin.Context) {
	userID := c.MustGet("userID").(int)
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var input postInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	postsMu.Lock()
	defer postsMu.Unlock()

	for i, post := range posts {
		if post.ID != postID {
			continue
		}
		if post.ID < localPostIDStart {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify external posts"})
			return
		}
		if post.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "not your post"})
			return
		}

		posts[i].Title = input.Title
		posts[i].Body = input.Body
		c.JSON(http.StatusOK, posts[i])
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
}

func deletePost(c *gin.Context) {
	userID := c.MustGet("userID").(int)
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	postsMu.Lock()
	defer postsMu.Unlock()

	for i, post := range posts {
		if post.ID != postID {
			continue
		}
		if post.ID < localPostIDStart {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify external posts"})
			return
		}
		if post.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "not your post"})
			return
		}

		posts = append(posts[:i], posts[i+1:]...)
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
}

func fetchAPIPosts(userID *int) ([]Post, error) {
	url := "https://jsonplaceholder.typicode.com/posts"
	if userID != nil {
		url = fmt.Sprintf("%s?userId=%d", url, *userID)
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var apiPosts []Post
	if err := json.Unmarshal(body, &apiPosts); err != nil {
		return nil, err
	}
	return apiPosts, nil
}

func getAllPosts(c *gin.Context) {
	apiPosts, err := fetchAPIPosts(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	postsMu.Lock()
	posts = append(posts, apiPosts...)
	postsMu.Unlock()

	postsMu.RLock()
	defer postsMu.RUnlock()
	c.JSON(http.StatusOK, posts)
}

func getPostsByUserID(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userID"})
		return
	}

	apiPosts, err := fetchAPIPosts(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	postsMu.RLock()
	localPosts := make([]Post, 0)
	for _, post := range posts {
		if post.UserID == userID {
			localPosts = append(localPosts, post)
		}
	}
	postsMu.RUnlock()

	result := append(apiPosts, localPosts...)
	c.JSON(http.StatusOK, result)
}
