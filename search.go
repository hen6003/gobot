package main

import (
	"io/ioutil"
	"net/http"
	"strings"
)

func search(query string) string {
	query = strings.Replace(query, " ", "+", -1)

	// Get the data
	resp, err := http.Get("https://www.youtube.com/results?search_query=" + query)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	file, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fileStr := string(file)

	i := strings.Index(fileStr, "videoId")

	return fileStr[i+10 : i+21]
}
