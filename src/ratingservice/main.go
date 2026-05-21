package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Ratings struct {
	Ratings []Rating `json:"ratings"`
}
type Rating struct {
	ID     string  `json:"id"`
	UserID int     `json:"user_id"`
	Score  float64 `json:"score"`
	Body   string  `json:"body"`
}

var ratings map[string]Rating

func main() {
	var err error
	ratings, err = loadRatings()
	if err != nil {
		panic(err)
	}

	router := gin.Default()
	router.GET("/healthz", healthz)
	router.GET("/ratings", getAllRatings)
	router.GET("/ratings/:id", getRatingByID)
	router.POST("/ratings/new", postRating)

	router.Run(":50001")
}

func healthz(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"message": "ok"})
}

func getAllRatings(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, ratings)
}

func getRatingByID(c *gin.Context) {
	id := c.Param("id")

	rating, ok := ratings[id]
	if ok == false {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "rating not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, rating)
}

func postRating(c *gin.Context) {
	var newRating Rating

	if err := c.BindJSON(&newRating); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	newRating.ID, _ = generateID(3)
	ratings[newRating.ID] = newRating

	c.IndentedJSON(http.StatusCreated, newRating)
}

func generateID(length int) (string, error) {
	// 6 bytes → 8 base64url chars, scale accordingly
	numBytes := (length*6)/8 + 1
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}
