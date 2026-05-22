package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type Ratings struct {
	Ratings []Rating `json:"ratings"`
}
type Rating struct {
	ID        string  `json:"id"`
	UserID    int     `json:"user_id"`
	Score     float64 `json:"score"`
	Body      string  `json:"body"`
	ProductID string  `json:"product_id"`
}

var ratings map[string]Rating

func main() {
	var err error
	ratings, err = loadRatings()
	if err != nil {
		panic(err)
	}

	router := gin.Default()
	router.GET("/_healthz", _healthz)
	router.GET("/ratings", getAllRatings)
	router.GET("/ratings/:id", getRatingsByID)
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
	c.IndentedJSON(http.StatusOK, ratings)
}

func getRatingsByID(c *gin.Context) {
	id := c.Param("id")

	rating, ok := ratings[id]
	if ok == false {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "rating not found"})
		return
	}

	c.IndentedJSON(http.StatusOK, rating)
}

func getRatingsByProductID(c *gin.Context) {
	productID := c.Param("product_id")

	productRatings := make(map[string]Rating)
	for rating, _ := range ratings {
		if ratings[rating].ProductID == productID {
			productRatings[rating] = ratings[rating]
		}
	}

	if len(productRatings) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "no ratings found for product"})
		return
	}

	c.IndentedJSON(http.StatusOK, productRatings)
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
