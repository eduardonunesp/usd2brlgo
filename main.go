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

type requestResult struct {
	Query struct {
		Count   int    `json:"count"`
		Created string `json:"created"`
		Lang    string `json:"lang"`
		Results string `json:"results"`
	} `json:"query"`
}

const url = "http://query.yahooapis.com/v1/public/yql?q=select%20*%20from%20html%20where%20url%3D%27www.google.com%2Ffinance%2Fconverter%3Fa%3D1%26from%3DUSD%26to%3DBRL%27%20and%20xpath%3D%27%2F%2F*%5B%40id%3D\"currency_converter_result\"%5D%2Fspan%2Ftext()%27&format=json&callback="

// Make the request and return the content of body
func checker() (string, error) {
	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Println("Error to close get request")
		}
	}()

	contents, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(contents), nil
}

// Parses the json and get only the value Results
func parseJSON(jsonResult string) string {
	log.Println("JSON REQUEST", jsonResult)
	result := new(requestResult)

	err := json.Unmarshal([]byte(jsonResult), result)

	if err != nil {
		log.Println(err)
	}

	log.Println(result)

	return result.Query.Results
}

// Checking the webservice each 30 minutes
func pool() chan string {
	ch := make(chan string)
	go func() {
		haveResponse := false
		requestJSON := func() {
			if res, err := checker(); err == nil {
				if jsonRes := parseJSON(res); jsonRes != "" {
					haveResponse = true
					ch <- jsonRes
				}
			}
		}

		for {
			requestJSON()
			if haveResponse {
				break
			} else {
				log.Println("No result try again ...")
			}
		}

		c := time.Tick(30 * time.Minute)
		for now := range c {
			log.Println("Updated at %v", now)
			requestJSON()
		}
	}()
	return ch
}

func main() {
	// Get the first value
	checkerpool := pool()
	latestResult := <-checkerpool

	// To deploy at heroku :D
	env := os.Getenv("GO_ENV")
	host := "127.0.0.1"
	port := 9000
	//token := ""

	if env == "PRODUCTION" {
		host = ""
		port, _ = strconv.Atoi(os.Getenv("PORT"))
		//token = os.Getenv("TOKEN")
	}

	address := fmt.Sprintf("%s:%d", host, port)
	log.Println("Ready to serve at", address)

	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		select {
		case latestResult := <-checkerpool:
			log.Println("Get result", latestResult)
		default:
			log.Println("Get cache value")
		}

		_, err := rw.Write([]byte(latestResult))

		if err != nil {
			log.Println(err)
		}
	})

	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatal("Failed to serve to address", address, err)
	}
}
