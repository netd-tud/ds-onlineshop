package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Ratings struct {
	Ratings []Rating `json:"ratings"`
}
type Rating struct {
	ID     int     `json:"id"`
	UserID int     `json:"user_id"`
	Score  float64 `json:"score"`
	Body   string  `json:"body"`
}

var ratings []Rating

func main() {
	var err error
	ratings, err = loadRatings()
	if err != nil {
		panic(err)
	}

	router := gin.Default()
	router.GET("/ratings", getAllRatings)
	router.GET("/ratings/:id", getRatingByID)
	router.POST("/ratings/new", postRating)

	router.Run(":50001")
}

func getAllRatings(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, ratings)
}

func getRatingByID(c *gin.Context) {
	strId := c.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "id needs to be convertable to int"})
		return
	}

	for _, a := range ratings {
		if a.ID == id {
			c.IndentedJSON(http.StatusOK, a)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "rating not found"})
}

func postRating(c *gin.Context) {
	var newRating Rating

	if err := c.BindJSON(&newRating); err != nil {
		return
	}

	ratings = append(ratings, newRating)
	c.IndentedJSON(http.StatusCreated, newRating)
}
