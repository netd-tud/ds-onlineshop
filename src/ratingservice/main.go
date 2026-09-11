package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"

	"github.com/gin-gonic/gin"
)

type ratingsForProduct struct {
	Ratings []rating `json:"ratings"`
	Average float32  `json:"average"`
}
type rating struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Score     float32 `json:"score"`
	Body      string  `json:"body"`
	ProductID string  `json:"product_id"`
}

var allRatings map[string]rating

func main() {
	var err error
	allRatings, err = loadRatings()
	if err != nil {
		panic(err)
	}

	router := gin.Default()
	router.GET("/_healthz", _healthz)
	router.GET("/ratings", getAllRatings)
	router.GET("/ratings/:id", getRatingByID)
	router.GET("/ratings/product/:product_id", getRatingsByProductID)
	router.POST("/ratings/new", postRating)

	port := os.Getenv("PORT")
	if port == "" {
		port = "50001"
	}

	router.Run(fmt.Sprintf(":%s", port))
}

func _healthz(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"message": "ok"})
}

func getAllRatings(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, allRatings)
}

// getRatingsByID handles the HTTP request to retrieve a specific rating by its unique identifier.
//
// It extracts the ID from the URL parameter, looks up the entry in the ratings store,
// and responds with either the serialized rating as indented JSON or an HTTP 404 Not Found error status.
func getRatingByID(c *gin.Context) {
	id := c.Param("id")

	rating, ok := allRatings[id]
	if ok == false {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "rating not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, rating)
}

func getRatingsByProductID(c *gin.Context) {
	productID := c.Param("product_id")

	productRatings := make(map[string]rating)
	totalScore := 0.0
	for rating, _ := range allRatings {
		if allRatings[rating].ProductID == productID {
			productRatings[rating] = allRatings[rating]
			totalScore += float64(allRatings[rating].Score)
		}
	}

	if len(productRatings) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "no ratings found for product"})
		return
	}

	averageScore := totalScore / float64(len(productRatings))
	r := ratingsForProduct{Ratings: slices.Collect(maps.Values(productRatings)), Average: float32(averageScore)}

	c.IndentedJSON(http.StatusOK, r)
}

func postRating(c *gin.Context) {
	var newRating rating

	if err := c.BindJSON(&newRating); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	newRating.ID, _ = generateID(3)
	allRatings[newRating.ID] = newRating

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
