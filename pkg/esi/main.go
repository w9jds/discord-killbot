package esi

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Client is a client for communication with the eve online api
type Client struct {
	baseURI string
	client  *http.Client
}

const baseURI = "https://esi.evetech.net"

// CreateClient creates a new instance of the Client
func CreateClient(httpClient *http.Client) *Client {
	return &Client{
		client: httpClient,
	}
}

func (esi Client) get(path string) ([]byte, error) {
	request, error := http.NewRequest("GET", baseURI+path, nil)
	if error != nil {
		return nil, error
	}

	return esi.do(request)
}

func (esi Client) post(path string, content []byte) ([]byte, error) {
	request, error := http.NewRequest("POST", baseURI+path, bytes.NewBuffer(content))
	if error != nil {
		return nil, error
	}

	request.Header.Add("User-Agent", "Aura Discord Bot - Chingy Chonga/Jeremy Shore - w9jds@live.com")
	request.Header.Add("Accept", "application/json")

	return esi.do(request)
}

func (esi Client) do(request *http.Request) ([]byte, error) {

	for i := 0; i < 3; i++ {
		response, error := esi.client.Do(request)
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

	return nil, errors.New("failed esi requests 3 times, gave up")
}
