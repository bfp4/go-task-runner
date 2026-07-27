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

// Post is a blog post as returned to clients. It matches the shape of the
// jsonplaceholder.typicode.com API so external and local posts look identical.
type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

// postInput is the request body accepted when creating or updating a post.
// The ID and owner are never client-supplied; they come from state and the
// X-User-ID header respectively.
type postInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// localPostIDStart is the first ID handed out to locally-created posts. IDs
// below it belong to the external API and are treated as read-only.
const localPostIDStart = 101

var (
	// posts holds only locally-created posts; external posts are fetched fresh
	// on each request and never stored. postsMu guards it since gin serves
	// requests concurrently. nextID is the ID for the next created post.
	posts   []Post
	postsMu sync.RWMutex
	nextID  = 101
)

func main() {
	r := gin.Default()

	// Public read endpoints.
	r.GET("/all-posts", getAllPosts)
	r.GET("/posts/:userID", getPostsByUserID)

	// Write endpoints require an X-User-ID header (see authMiddleware).
	authorized := r.Group("/", authMiddleware())
	authorized.POST("/create-post", createPost)
	authorized.PUT("/posts/:id", updatePost)
	authorized.DELETE("/posts/:id", deletePost)

	r.Run(":8080")
}

// authMiddleware is a stand-in for real authentication: it trusts whatever
// user ID the client sends in the X-User-ID header and stashes it in the
// context for handlers to read.
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

// createPost stores a new post owned by the requesting user and returns it
// with its freshly assigned ID.
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

// updatePost edits the title/body of a local post. A user may only edit their
// own posts, and external posts (ID < localPostIDStart) cannot be edited.
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

// deletePost removes a local post. Same ownership and external-post rules as
// updatePost apply.
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

// fetchAPIPosts pulls posts from the external jsonplaceholder API, optionally
// filtered to a single user.
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

// getAllPosts returns every external post followed by every locally-created one.
func getAllPosts(c *gin.Context) {
	apiPosts, err := fetchAPIPosts(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return external posts plus locally-created ones without persisting the
	// external posts, otherwise every call would accumulate duplicates.
	postsMu.RLock()
	result := make([]Post, 0, len(apiPosts)+len(posts))
	result = append(result, apiPosts...)
	result = append(result, posts...)
	postsMu.RUnlock()

	c.JSON(http.StatusOK, result)
}

// getPostsByUserID returns the external and local posts belonging to one user.
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

	// Filter the local store down to this user's posts.
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
