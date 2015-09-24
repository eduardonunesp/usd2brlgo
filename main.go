package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type RequestResult struct {
	Query struct {
		Count   int    `json:"count"`
		Created string `json:"created"`
		Lang    string `json:"lang"`
		Results string `json:"results"`
	} `json:"query"`
}

const URL = "http://query.yahooapis.com/v1/public/yql?q=select%20*%20from%20html%20where%20url%3D%27www.google.com%2Ffinance%2Fconverter%3Fa%3D1%26from%3DUSD%26to%3DBRL%27%20and%20xpath%3D%27%2F%2F*%5B%40id%3D\"currency_converter_result\"%5D%2Fspan%2Ftext()%27&format=json&callback="

// Make the request and return the content of body
func Checker() (string, error) {
	if resp, err := http.Get(URL); err != nil {
		return "", err
	} else {
		defer resp.Body.Close()
		if contents, err := ioutil.ReadAll(resp.Body); err != nil {
			return "", err
		} else {
			return string(contents), nil
		}
	}
}

// Parses the json and get only the value Results
func ParseJSON(jsonResult string) string {
	log.Println("JSON REQUEST", jsonResult)
	result := new(RequestResult)

	err := json.Unmarshal([]byte(jsonResult), result)

	if err != nil {
		log.Println(err)
	}

	log.Println(result)

	return result.Query.Results
}

// Checking the webservice each 30 minutes
func Pool() chan string {
	ch := make(chan string)
	go func() {
		for {
			if res, err := Checker(); err == nil {
				if jsonRes := ParseJSON(res); jsonRes != "" {
					ch <- jsonRes
					time.Sleep(time.Minute * 30)
				}
			}
		}
	}()
	return ch
}

func main() {
	// Get the first value
	checkerPool := Pool()
	latestResult := <-checkerPool

	// To deploy at heroku :D
	env := os.Getenv("GO_ENV")
	host := "127.0.0.1"
	port := 9000

	if env == "PRODUCTION" {
		host = ""
		port, _ = strconv.Atoi(os.Getenv("PORT"))
	}

	address := fmt.Sprintf("%s:%d", host, port)
	log.Println("Ready to serve at", address)

	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		log.Println(len(checkerPool))

		// Check if chan res a value waiting
		if len(checkerPool) >= 1 {
			latestResult = <-checkerPool
		}

		rw.Write([]byte(latestResult))
	})

	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatal("Failed to serve to address", address, err)
	}
}
