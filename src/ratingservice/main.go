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

// ratingsForProduct represents a collection of individual rating entries
// and their computed overall average score, associated to a product.
type ratingsForProduct struct {
	Ratings []rating `json:"ratings"`
	Average float32  `json:"average"`
}

// rating represents a data structure for an individual product review,
// including its numerical score, textual feedback, and associated user and product identifiers.
type rating struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Score     float32 `json:"score"`
	Body      string  `json:"body"`
	ProductID string  `json:"product_id"`
}

// allRatings is a map of rating entries accessed by their rating ID
var allRatings map[string]rating

// main initializes and boots the ratings microservice HTTP server.
//
// It loads existing product ratings into memory, configures the Gin web router with health check
// and rating endpoints, and starts listening for incoming HTTP requests on the designated port.
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

// _healthz handles the HTTP liveness and readiness probe requests.
//
// It returns a simple JSON status payload confirming that the service is healthy.
func _healthz(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"message": "ok"})
}

// getAllRatings handles the HTTP request to retrieve all stored ratings.
//
// It returns the complete ratings collection serialized as indented JSON with an HTTP 200 OK status.
func getAllRatings(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, allRatings)
}

// getRatingByID handles the HTTP request to retrieve a specific rating by its unique identifier.
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

// getRatingsByProductID handles the HTTP request to retrieve all ratings associated with a specific product.
//
// It filters the ratings collection by the given product ID, computes the overall average score,
// and returns the matching ratings and aggregate score as indented JSON or an HTTP 404 Not Found error status.
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

// postRating handles the HTTP request to create a new product rating.
//
// It parses and validates the incoming JSON request body into a rating structure,
// generates a unique short identifier for the review, stores it in memory,
// and responds with the newly created rating and an HTTP 201 Created status.
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

// generateID generates a cryptographically secure, URL-safe random string identifier
// of the specified length using base64 encoding.
func generateID(length int) (string, error) {
	// 6 bytes → 8 base64url chars, scale accordingly
	numBytes := (length*6)/8 + 1
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}
