package main

import (
	// "io"
	"io/ioutil"
	"log"
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
		log.Println(err)
	}
	fileStr := string(file)

	i := strings.Index(fileStr, "videoId")

	return fileStr[i+10 : i+21]
}

func imgSearch(query string) string {
	query = strings.Replace(query, " ", "+", -1)

	resp, err := http.Get("https://pixabay.com/api/?key=" + key + "&q=" + query)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	file, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	fileStr := string(file)

	i := strings.Index(fileStr, "webformatURL")

	fileStr = fileStr[i+15:]

	i = strings.Index(fileStr, "\"")

	fileStr = fileStr[:i]

	return fileStr
}
