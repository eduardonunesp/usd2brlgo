package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	cacheUpdateInterval = 1 * time.Minute
	url                 = "https://economia.awesomeapi.com.br/last/USD-BRL"
)

type requestResult struct {
	Query struct {
		Results string `json:"bid"`
	} `json:"USDBRL"`
}

// Make the request and return the content of body
func checker() ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Println("Error to close get request")
		}
	}()

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

// Parses the json and get only the value Results
func parseJSON(data []byte) []byte {
	var result requestResult

	log.Println("JSON REQUEST", string(data))

	err := json.Unmarshal(data, &result)
	if err != nil {
		log.Println(err)
		return nil
	}

	log.Println(result)

	return []byte(result.Query.Results)
}

// Checking the webservice each 30 minutes
func pool() chan []byte {
	ch := make(chan []byte)
	go func() {
		requestJSON := func() {
			log.Printf("Checking the webservice %s", url)
			if res, err := checker(); err == nil {
				if result := parseJSON(res); result != nil {
					ch <- result
				}
			}
		}

		requestJSON()

		c := time.Tick(cacheUpdateInterval)
		for now := range c {
			log.Printf("Updated at %v", now)
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
	port := "9000"

	if env == "PRODUCTION" {
		host = ""
		port = os.Getenv("PORT")
	}

	address := fmt.Sprintf("%s:%s", host, port)
	log.Println("Ready to serve at", address)

	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		select {
		case latestResult = <-checkerpool:
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
