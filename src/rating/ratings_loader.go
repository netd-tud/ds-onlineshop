package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func loadRatings() ([]Rating, error) {
	jsonFile, err := os.Open("ratings.json")

	if err != nil {
		fmt.Println("Error opening File: ", err)
		return nil, err
	}

	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(jsonFile)

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Error reading file: ", err)
		return nil, err
	}

	var wrapper struct {
		Ratings []Rating `json:"ratings"`
	}

	err = json.Unmarshal(byteValue, &wrapper)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return nil, err
	}

	return wrapper.Ratings, nil
}
