package zkb

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Client is an interface to interact with the zkb api
type Client struct {
	client *http.Client
}

// CreateClient creates a new zkb client
func CreateClient(httpClient *http.Client) *Client {
	return &Client{
		client: httpClient,
	}
}

func (zkb Client) get(baseURI string, path string) ([]byte, error) {
	request, error := http.NewRequest("GET", baseURI+path, nil)
	if error != nil {
		return nil, error
	}

	for i := 0; i < 3; i++ {
		response, error := zkb.client.Do(request)
		if error != nil {
			log.Println(error)
			continue
		} else if response.StatusCode < 200 || response.StatusCode > 299 {
			message, error := ioutil.ReadAll(response.Body)
			if error != nil {
				log.Println(error)
				time.Sleep(5 * time.Second)
				continue
			} else {
				log.Println(string(message))
				time.Sleep(5 * time.Second)
				continue
			}
		} else {
			return ioutil.ReadAll(response.Body)
		}
	}

	return nil, errors.New("failed zkb requests 3 times, gave up")
}
