package main

import (
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

	resp, err := http.Get("https://yandex.com/images/search?text=" + query)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	file, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	fileStr := string(file)

	i := strings.Index(fileStr, "pos=0")

	fileStr = fileStr[i+18:]

	i = strings.Index(fileStr, "&")

	fileStr = fileStr[:i]

	fileStr = strings.Replace(fileStr, "%2F", "/", -1)
	fileStr = strings.Replace(fileStr, "%3A", ":", -1)
	fileStr = strings.Replace(fileStr, "%3F", "?", -1)
	fileStr = strings.Replace(fileStr, "%3D", "=", -1)

	return fileStr
}
